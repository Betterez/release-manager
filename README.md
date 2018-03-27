Release manager (rmg)
========================
rmg is a tool to pull instances from a release target to a production target.
You can select your environment, and path type.

What's required
------------------
`rmg` uses the aws tagging system to determined which target groups to load, which are target and which are the source.

**Source** will be any target group with the `Release` tag set to `yes`.

**target** will be any target group with the `Release` tag set to `no`.

There are three other parameters:
1. Environment
2. Path
3. Type

These are required to find the right target groups.

Usage:
--------------
rmg --env=environment name --path=path value --elb-type=elb type
