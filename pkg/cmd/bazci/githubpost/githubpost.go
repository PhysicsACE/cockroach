// Copyright 2016 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Command github-post parses the JSON-formatted output from a Go test session,
// as generated by either 'go test -json' or './pkg.test | go tool test2json -t',
// and posts issues for any failed tests to GitHub. If there are no failed
// tests, it assumes that there was a build error and posts the entire log to
// GitHub.

package githubpost

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	buildutil "github.com/cockroachdb/cockroach/pkg/build/util"
	"github.com/cockroachdb/cockroach/pkg/cmd/bazci/githubpost/issues"
	"github.com/cockroachdb/cockroach/pkg/internal/codeowners"
	"github.com/cockroachdb/cockroach/pkg/internal/team"
	"github.com/cockroachdb/errors"
)

const (
	pkgEnv  = "PKG"
	unknown = "(unknown)"
)

type packageAndTest struct {
	packageName, testName string
}

type fileAndLine struct {
	filename, linenum string
}

// This should be set to some non-nil map in test scenarios. This is used to
// mock out the results of the getFileLine function.
var fileLineForTesting map[packageAndTest]fileAndLine

type Formatter func(context.Context, Failure) (issues.IssueFormatter, issues.PostRequest)
type FailurePoster func(context.Context, Failure) error

func DefaultFormatter(ctx context.Context, f Failure) (issues.IssueFormatter, issues.PostRequest) {
	teams := getOwner(ctx, f.packageName, f.testName)
	repro := fmt.Sprintf("./dev test ./pkg/%s --race --stress -f %s",
		trimPkg(f.packageName), f.testName)

	var projColID int
	var mentions []string
	var labels []string
	if os.Getenv("SKIP_LABEL_TEST_FAILURE") == "" {
		labels = append(labels, issues.DefaultLabels...)
	}
	if len(teams) > 0 {
		projColID = teams[0].TriageColumnID
		for _, team := range teams {
			mentions = append(mentions, "@"+string(team.Name()))
			if team.Label != "" {
				labels = append(labels, team.Label)
			}
		}
	}
	return issues.UnitTestFormatter, issues.PostRequest{
		TestName:        f.testName,
		PackageName:     f.packageName,
		Message:         f.testMessage,
		Artifacts:       "/", // best we can do for unit tests
		HelpCommand:     issues.UnitTestHelpCommand(repro),
		MentionOnCreate: mentions,
		ProjectColumnID: projColID,
		Labels:          labels,
	}
}

func DefaultIssueFilerFromFormatter(
	formatter Formatter,
) func(ctx context.Context, f Failure) error {
	opts := issues.DefaultOptionsFromEnv()
	return func(ctx context.Context, f Failure) error {
		fmter, req := formatter(ctx, f)
		if stress := os.Getenv("COCKROACH_NIGHTLY_STRESS"); stress != "" {
			if req.ExtraParams == nil {
				req.ExtraParams = make(map[string]string)
			}
			req.ExtraParams["stress"] = "true"
		}
		_, err := issues.Post(ctx, log.Default(), fmter, req, opts)
		return err
	}

}

func getFailurePosterFromFormatterName(formatterName string) FailurePoster {
	var reqFromFailure Formatter
	switch formatterName {
	case "pebble-metamorphic":
		reqFromFailure = formatPebbleMetamorphicIssue
	default:
		reqFromFailure = DefaultFormatter
	}
	return DefaultIssueFilerFromFormatter(reqFromFailure)
}

// PostFromJSON parses the JSON-formatted output from a Go test session,
// as generated by either 'go test -json' or './pkg.test | go tool test2json -t',
// and posts issues for any failed tests to GitHub. If there are no failed
// tests, it assumes that there was a build error and posts the entire log to
// GitHub.
func PostFromJSON(formatterName string, in io.Reader) {
	ctx := context.Background()
	fileIssue := getFailurePosterFromFormatterName(formatterName)
	if err := listFailuresFromJSON(ctx, in, fileIssue); err != nil {
		log.Println(err) // keep going
	}
}

// PostFromTestXMLWithFormatterName consumes a Bazel-style `test.xml` stream
// and posts issues for any failed tests to GitHub. If there are no failed
// tests, it does nothing. Unlike PostFromTestXMLWithFailurePoster, it takes a
// formatter name.
func PostFromTestXMLWithFormatterName(formatterName string, testXml buildutil.TestSuites) error {
	ctx := context.Background()
	fileIssue := getFailurePosterFromFormatterName(formatterName)
	return PostFromTestXMLWithFailurePoster(ctx, fileIssue, testXml)
}

// PostFromTestXMLWithFailurePoster consumes a Bazel-style `test.xml` stream and posts
// issues for any failed tests to GitHub. If there are no failed tests, it does
// nothing.
func PostFromTestXMLWithFailurePoster(
	ctx context.Context, fileIssue FailurePoster, testXml buildutil.TestSuites,
) error {
	return listFailuresFromTestXML(ctx, testXml, fileIssue)
}

type Failure struct {
	title       string
	packageName string
	testName    string
	testMessage string
}

// This struct is described in the test2json documentation.
// https://golang.org/cmd/test2json/
type testEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
	Time    time.Time // encodes as an RFC3339-format string
	Elapsed float64   // seconds
}

type scopedTest struct {
	pkg  string
	name string
}

func scoped(te testEvent) scopedTest {
	if te.Package == "" {
		return scopedTest{pkg: mustPkgFromEnv(), name: te.Test}
	}
	return scopedTest{pkg: te.Package, name: te.Test}
}

func mustPkgFromEnv() string {
	packageName := os.Getenv(pkgEnv)
	if packageName == "" {
		panic(errors.Errorf("package name environment variable %s is not set", pkgEnv))
	}
	return packageName
}

func maybeEnv(envKey, defaultValue string) string {
	v := os.Getenv(envKey)
	if v == "" {
		return defaultValue
	}
	return v
}

func shortPkg() string {
	packageName := maybeEnv(pkgEnv, "unknown")
	return trimPkg(packageName)
}

func trimPkg(pkg string) string {
	return strings.TrimPrefix(pkg, issues.CockroachPkgPrefix)
}

func listFailuresFromJSON(
	ctx context.Context, input io.Reader, fileIssue func(context.Context, Failure) error,
) error {
	// Tests that took less than this are not even considered for slow test
	// reporting. This is so that we protect against large number of
	// programmatically-generated subtests.
	const shortTestFilterSecs float64 = 0.5
	var timeoutMsg = "panic: test timed out after"

	var packageOutput bytes.Buffer

	// map  from test name to list of events (each log line is an event, plus
	// start and pass/fail events).
	// Tests/events are "outstanding" until we see a final pass/fail event.
	// Because of the way the go test runner prints output, in case a subtest times
	// out or panics, we don't get a pass/fail event for sibling and ancestor
	// tests. Those tests will remain "outstanding" and will be ignored for the
	// purpose of issue reporting.
	outstandingOutput := make(map[scopedTest][]testEvent)
	failures := make(map[scopedTest][]testEvent)
	var slowPassEvents []testEvent
	var slowFailEvents []testEvent

	// init is true for the preamble of the input before the first "run" test
	// event.
	init := true
	// trustTimestamps will be set if we don't find a marker suggesting that the
	// input comes from a stress run. In that case, stress prints all its output
	// at once (for a captured failed test run), so the test2json timestamps are
	// meaningless.
	trustTimestamps := true
	// elapsedTotalSec accumulates the time spent in all tests, passing or
	// failing. In case the input comes from a stress run, this will be used to
	// deduce the duration of a timed out test.
	var elapsedTotalSec float64
	// Will be set if the last test timed out.
	var timedOutCulprit scopedTest
	var timedOutEvent testEvent
	var curTestStart time.Time
	var last scopedTest
	var lastEvent testEvent
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		var te testEvent
		{
			line := scanner.Text() // has no EOL marker
			if len(line) <= 2 || line[0] != '{' || line[len(line)-1] != '}' {
				// This line is not test2json output, skip it. This can happen if
				// whatever feeds our input has some extra non-JSON lines such as
				// would happen with `make` invocations.
				continue
			}
			if err := json.Unmarshal([]byte(line), &te); err != nil {
				return errors.Wrapf(err, "unable to parse %q", line)
			}
		}
		lastEvent = te

		if te.Test != "" {
			init = false
		}
		if init && strings.Contains(te.Output, "-exec 'stress '") {
			trustTimestamps = false
		}
		if timedOutCulprit.name == "" && te.Elapsed > 0 {
			// We don't count subtests as those are counted in the parent.
			if split := strings.SplitN(te.Test, "/", 2); len(split) == 1 {
				elapsedTotalSec += te.Elapsed
			}
		}

		if timedOutCulprit.name == te.Test && te.Elapsed != 0 {
			te.Elapsed = timedOutEvent.Elapsed
		}

		// Events for the overall package test do not set Test.
		if len(te.Test) > 0 {
			switch te.Action {
			case "run":
				last = scoped(te)
				if trustTimestamps {
					curTestStart = te.Time
				}
			case "output":
				key := scoped(te)
				outstandingOutput[key] = append(outstandingOutput[key], te)
				if strings.Contains(te.Output, timeoutMsg) {
					timedOutCulprit = key

					// Fill in the Elapsed field for a timeout event.
					// As of go1.11, the Elapsed field is bogus for fail events for timed
					// out tests, so we do our own computation.
					// See https://github.com/golang/go/issues/27568
					//
					// Also, if the input is coming from stress, there will not even be a
					// fail event for the test, so the Elapsed field computed here will be
					// useful.
					if trustTimestamps {
						te.Elapsed = te.Time.Sub(curTestStart).Seconds()
					} else {
						// If we don't trust the timestamps, then we compute the test's
						// duration by subtracting all the durations that we've seen so far
						// (which we do trust to some extent). Note that this is not
						// entirely accurate, since there's no information about the
						// duration about sibling subtests which may have run. And further
						// note that it doesn't work well at all for small timeouts because
						// the resolution that the test durations have is just tens of
						// milliseconds, so many quick tests are rounded of to a duration of
						// 0.
						re := regexp.MustCompile(`panic: test timed out after (\d*(?:\.\d*)?)(.)`)
						matches := re.FindStringSubmatch(te.Output)
						if matches == nil {
							log.Printf("failed to parse timeout message: %s", te.Output)
							te.Elapsed = -1
						} else {
							dur, err := strconv.ParseFloat(matches[1], 64)
							if err != nil {
								return err
							}
							if matches[2] == "m" {
								// minutes to seconds
								dur *= 60
							} else if matches[2] != "s" {
								return fmt.Errorf("unexpected time unit in: %s", te.Output)
							}
							te.Elapsed = dur - elapsedTotalSec
						}
					}
					timedOutEvent = te
				}
			case "pass", "skip":
				if timedOutCulprit.name != "" {
					// NB: we used to do this:
					//   panic(fmt.Sprintf("detected test timeout but test seems to have passed (%+v)", te))
					// but it would get hit. There is no good way to
					// blame a timeout on a particular test. We should probably remove this
					// logic in the first place, but for now make sure we don't panic as
					// a result of it.
					timedOutCulprit = scopedTest{}
				}
				delete(outstandingOutput, scoped(te))
				if te.Elapsed > shortTestFilterSecs {
					// We ignore subtests; their time contributes to the parent's.
					if !strings.Contains(te.Test, "/") {
						slowPassEvents = append(slowPassEvents, te)
					}
				}
			case "fail":
				key := scoped(te)
				// Record slow tests. We ignore subtests; their time contributes to the
				// parent's. Except the timed out (sub)test, for which the parent (if
				// any) is not going to appear in the report because there's not going
				// to be a pass/fail event for it.
				if !strings.Contains(te.Test, "/") || timedOutCulprit == key {
					slowFailEvents = append(slowFailEvents, te)
				}
				// Move the test to the failures collection unless the test timed out.
				// We have special reporting for timeouts below.
				if timedOutCulprit != key {
					failures[key] = outstandingOutput[key]
				}
				delete(outstandingOutput, key)
			}
		} else if te.Action == "output" {
			// Output was outside the context of a test. This consists mostly of the
			// preamble and epilogue that Make outputs, but also any log messages that
			// are printed by a test binary's main function.
			packageOutput.WriteString(te.Output)
		}
	}

	// On timeout, we might or might not have gotten a fail event for the timed
	// out test (we seem to get one when processing output from a test binary run,
	// but not when processing the output of `stress`, which adds some lines at
	// the end). If we haven't gotten a fail event, the test's output is still
	// outstanding and the test is not registered in the slowFailEvents
	// collection. The timeout handling code below relies on slowFailEvents not
	// being empty though, so we'll process the test here.
	if timedOutCulprit.name != "" {
		if _, ok := outstandingOutput[timedOutCulprit]; ok {
			slowFailEvents = append(slowFailEvents, timedOutEvent)
			delete(outstandingOutput, timedOutCulprit)
		}
	} else {
		// If we haven't received a final event for the last test, then a
		// panic/log.Fatal must have happened. Consider it failed.
		// Note that because of https://github.com/golang/go/issues/27582 there
		// might be other outstanding tests; we ignore those.
		if _, ok := outstandingOutput[last]; ok {
			log.Printf("found outstanding output. Considering last test failed: %s", last)
			failures[last] = outstandingOutput[last]
		}
	}

	// test2json always puts a fail event last unless it sees a big pass message
	// from the test output.
	if lastEvent.Action == "fail" && len(failures) == 0 && timedOutCulprit.name == "" {
		// If we couldn't find a failing Go test, assume that a failure occurred
		// before running Go and post an issue about that.
		err := fileIssue(ctx, Failure{
			title:       fmt.Sprintf("%s: package failed", shortPkg()),
			packageName: maybeEnv(pkgEnv, "unknown"),
			testName:    unknown,
			testMessage: packageOutput.String(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to post issue")
		}
	}

	if err := processFailures(ctx, fileIssue, failures); err != nil {
		return err
	}

	// Sort slow tests descendingly by duration.
	sort.Slice(slowPassEvents, func(i, j int) bool {
		return slowPassEvents[i].Elapsed > slowPassEvents[j].Elapsed
	})
	sort.Slice(slowFailEvents, func(i, j int) bool {
		return slowFailEvents[i].Elapsed > slowFailEvents[j].Elapsed
	})

	report := genSlowTestsReport(slowPassEvents, slowFailEvents)
	if err := writeSlowTestsReport(report); err != nil {
		log.Printf("failed to create slow tests report: %s", err)
	}

	// If the run timed out, file an issue. A couple of cases:
	// 1) If the test that was running when the package timed out is the longest
	// test, then we blame it. The common case is the test deadlocking - it would
	// have run forever.
	// 2) Otherwise, we don't blame anybody in particular. We file a generic issue
	// listing the package name containing the report of long-running tests.
	if timedOutCulprit.name != "" {
		slowest := slowFailEvents[0]
		if len(slowPassEvents) > 0 && slowPassEvents[0].Elapsed > slowest.Elapsed {
			slowest = slowPassEvents[0]
		}

		culpritOwner := fmt.Sprintf("%v", getOwner(ctx, timedOutCulprit.pkg, timedOutCulprit.name))
		slowEvents := append(slowFailEvents, slowPassEvents...)
		// The predicate determines if the union of the slow events is owned by the _same_ team(s) as timedOutCulprit.
		hasSameOwner := func() bool {
			for _, slowEvent := range slowEvents {
				scopedEvent := scoped(slowEvent)
				owner := fmt.Sprintf("%v", getOwner(ctx, scopedEvent.pkg, scopedEvent.name))

				if culpritOwner != owner {
					log.Printf("%v has a different owner: %s;bailing out...\n", scopedEvent, owner)
					return false
				}
			}
			return true
		}
		if timedOutCulprit == scoped(slowest) || hasSameOwner() {
			// The test that was running when the timeout hit is either the one that ran for
			// the longest time or all other tests share the same owner.
			// The test that was running when the timeout hit is the one that ran for
			// the longest time.
			log.Printf("timeout culprit found: %s\n", timedOutCulprit.name)
			err := fileIssue(ctx, Failure{
				title:       fmt.Sprintf("%s: %s timed out", trimPkg(timedOutCulprit.pkg), timedOutCulprit.name),
				packageName: timedOutCulprit.pkg,
				testName:    timedOutCulprit.name,
				testMessage: report,
			})
			if err != nil {
				return errors.Wrap(err, "failed to post issue")
			}
		} else {
			log.Printf("timeout culprit not found\n")
			// TODO(irfansharif): These are assigned to nobody given our lack of
			// a story around #51653. It'd be nice to be able to go from pkg
			// name to team-name, and be able to assign to a specific team.
			err := fileIssue(ctx, Failure{
				title:       fmt.Sprintf("%s: package timed out", shortPkg()),
				packageName: maybeEnv(pkgEnv, "unknown"),
				testName:    unknown,
				testMessage: report,
			})
			if err != nil {
				return errors.Wrap(err, "failed to post issue")
			}
		}
	}

	return nil
}

func listFailuresFromTestXML(
	ctx context.Context, suites buildutil.TestSuites, fileIssue func(context.Context, Failure) error,
) error {
	failures := make(map[scopedTest][]testEvent)
	for _, suite := range suites.Suites {
		pkg := suite.Name
		for _, testCase := range suite.TestCases {
			var result *buildutil.XMLMessage
			if testCase.Failure != nil {
				result = testCase.Failure
			} else if testCase.Error != nil {
				result = testCase.Error
			}
			if result != nil {
				key := scopedTest{
					pkg:  pkg,
					name: testCase.Name,
				}
				elapsed := 0.0
				if testCase.Time != "" {
					var err error
					elapsed, err = strconv.ParseFloat(testCase.Time, 64)
					if err != nil {
						fmt.Printf("couldn't parse time %s as float64: %+v\n", testCase.Time, err)
					}
				}
				event := testEvent{
					Action:  "fail",
					Package: pkg,
					Test:    testCase.Name,
					Output:  result.Contents,
					Elapsed: elapsed,
				}
				failures[key] = append(failures[key], event)
			}
		}
	}

	if len(failures) > 0 {
		return processFailures(ctx, fileIssue, failures)
	}
	return nil
}

func processFailures(
	ctx context.Context,
	fileIssue func(context.Context, Failure) error,
	failures map[scopedTest][]testEvent,
) error {
	for test, testEvents := range failures {
		if split := strings.SplitN(test.name, "/", 2); len(split) == 2 {
			parentTest, subTest := scopedTest{pkg: test.pkg, name: split[0]}, scopedTest{pkg: test.pkg, name: split[1]}
			log.Printf("consolidating failed subtest %q into parent test %q", subTest.name, parentTest.name)
			failures[parentTest] = append(failures[parentTest], testEvents...)
			delete(failures, test)
		} else {
			log.Printf("failed parent test %q (no subtests)", test.name)
			if _, ok := failures[test]; !ok {
				return errors.AssertionFailedf("expected %q in 'failures'", test.name)
			}
		}
	}
	// Sort the failed tests to make the unit tests for this script deterministic.
	var failedTestNames []scopedTest
	for name := range failures {
		failedTestNames = append(failedTestNames, name)
	}
	sort.Slice(failedTestNames, func(i, j int) bool {
		return fmt.Sprint(failedTestNames[i]) < fmt.Sprint(failedTestNames[j])
	})
	for _, test := range failedTestNames {
		testEvents := failures[test]
		var outputs []string
		for _, testEvent := range testEvents {
			outputs = append(outputs, testEvent.Output)
		}
		err := fileIssue(ctx, Failure{
			title:       fmt.Sprintf("%s: %s failed", trimPkg(test.pkg), test.name),
			packageName: test.pkg,
			testName:    test.name,
			testMessage: strings.Join(outputs, ""),
		})
		if err != nil {
			return errors.Wrap(err, "failed to post issue")
		}
	}

	return nil
}

func genSlowTestsReport(slowPassingTests, slowFailingTests []testEvent) string {
	var b strings.Builder
	b.WriteString("Slow failing tests:\n")
	for i, te := range slowFailingTests {
		if i == 20 {
			break
		}
		fmt.Fprintf(&b, "%s - %.2fs\n", te.Test, te.Elapsed)
	}
	if len(slowFailingTests) == 0 {
		fmt.Fprint(&b, "<none>\n")
	}

	b.WriteString("\nSlow passing tests:\n")
	for i, te := range slowPassingTests {
		if i == 20 {
			break
		}
		fmt.Fprintf(&b, "%s - %.2fs\n", te.Test, te.Elapsed)
	}
	if len(slowPassingTests) == 0 {
		fmt.Fprint(&b, "<none>\n")
	}
	return b.String()
}

func writeSlowTestsReport(report string) error {
	return os.WriteFile("artifacts/slow-tests-report.txt", []byte(report), 0644)
}

// getFileLine returns the file (relative to repo root) and line for the given test.
// The package name is assumed relative to the repo root as well, i.e. pkg/foo/bar.
func getFileLine(
	ctx context.Context, packageName, testName string,
) (_filename string, _linenum string, _ error) {
	if fileLineForTesting != nil {
		res, ok := fileLineForTesting[packageAndTest{packageName: packageName, testName: testName}]
		if ok {
			return res.filename, res.linenum, nil
		}
		return "", "", errors.Newf("could not find testing line:num for %s.%s", packageName, testName)
	}
	// Search the source code for the email address of the last committer to touch
	// the first line of the source code that contains testName. Then, ask GitHub
	// for the GitHub username of the user with that email address by searching
	// commits in cockroachdb/cockroach for commits authored by the address.
	subtests := strings.Split(testName, "/")
	testName = subtests[0]
	packageName = strings.TrimPrefix(packageName, "github.com/cockroachdb/cockroach/")
	for {
		if !strings.Contains(packageName, "pkg") {
			return "", "", errors.Newf("could not find test %s", testName)
		}
		cmd := exec.Command(`/bin/bash`, `-c`,
			fmt.Sprintf(`cd "$(git rev-parse --show-toplevel)" && git grep -n 'func %s(' '%s/*_test.go'`,
				testName, packageName))
		// This command returns output such as:
		// ../ccl/storageccl/export_test.go:31:func TestExportCmd(t *testing.T) {
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("couldn't find test %s in %s: %s %+v\n",
				testName, packageName, string(out), err)
			packageName = filepath.Dir(packageName)
			continue
		}
		re := regexp.MustCompile(`(.*):(.*):`)
		// The first 2 :-delimited fields are the filename and line number.
		matches := re.FindSubmatch(out)
		if matches == nil {
			fmt.Printf("couldn't find filename/line number for test %s in %s: %s %+v\n",
				testName, packageName, string(out), err)
			packageName = filepath.Dir(packageName)
			continue
		}
		return string(matches[1]), string(matches[2]), nil
	}
}

// getOwner looks up the file containing the given test and returns
// the owning teams. It does not return
// errors, but instead simply returns what it can.
// In case no owning team is found, "test-eng" team is returned.
func getOwner(ctx context.Context, packageName, testName string) (_teams []team.Team) {
	filename, _, err := getFileLine(ctx, packageName, testName)
	if err != nil {
		log.Printf("error getting file:line for %s.%s: %s", packageName, testName, err)
		// Let's continue so that we can assign the "catch-all" owner.
	}
	co, err := codeowners.DefaultLoadCodeOwners()
	if err != nil {
		log.Printf("loading codeowners: %s", err)
		return nil
	}
	match := co.Match(filename)

	if match == nil {
		// N.B. if no owning team is found, we default to 'test-eng'. This should be a rare exception rather than the rule.
		testEng := co.GetTeamForAlias("cockroachdb/test-eng")
		if testEng.Name() == "" {
			log.Fatalf("test-eng team could not be found in TEAMS.yaml")
		}
		log.Printf("assigning %s.%s to 'test-eng' as catch-all", packageName, testName)
		match = []team.Team{testEng}
	}
	return match
}

func formatPebbleMetamorphicIssue(
	ctx context.Context, f Failure,
) (issues.IssueFormatter, issues.PostRequest) {
	var repro string
	{
		const seedHeader = "===== SEED =====\n"
		i := strings.Index(f.testMessage, seedHeader)
		if i != -1 {
			s := f.testMessage[i+len(seedHeader):]
			s = strings.TrimSpace(s)
			s = strings.TrimSpace(s[:strings.Index(s, "\n")])
			repro = fmt.Sprintf(`go test -tags 'invariants' -exec 'stress -p 1' `+
				`-timeout 0 -test.v -run '%s$' ./internal/metamorphic -seed %s -ops "uniform:5000-10000"`, f.testName, s)
		}
	}
	return issues.UnitTestFormatter, issues.PostRequest{
		TestName:    f.testName,
		PackageName: f.packageName,
		Message:     f.testMessage,
		Artifacts:   "meta",
		HelpCommand: issues.ReproductionCommandFromString(repro),
		Labels:      append([]string{"metamorphic-failure"}, issues.DefaultLabels...),
	}
}

// PostGeneralFailure posts a "general" GitHub issue that does not correspond
// to any particular failed test, etc. These will generally be build failures
// that prevent any tests from having run. In this case we have very little
// insight into what caused the build failure and can't properly assign owners,
// so a general issue is filed against test-eng in this case.
func PostGeneralFailure(formatterName, logs string) {
	fileIssue := getFailurePosterFromFormatterName(formatterName)
	postGeneralFailureImpl(logs, fileIssue)
}

func postGeneralFailureImpl(logs string, fileIssue func(context.Context, Failure) error) {
	ctx := context.Background()
	err := fileIssue(ctx, Failure{
		title:       "unexpected build failure",
		testMessage: logs,
	})
	if err != nil {
		log.Println(err) // keep going
	}

}
