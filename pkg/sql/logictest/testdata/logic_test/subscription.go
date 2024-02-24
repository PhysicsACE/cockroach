query T
SELECT ARRAY['a', 'b', 'c'][1:2]
----
{a,b}