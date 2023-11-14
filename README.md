# Udger Golang

This package reads in memory all the database from [Udger](https://udger.com/) and lets you lookup for user agent's metadata. The parsing only relies on the golang standard library regex. Only the **Udger Data v3 - Sqlite3 format** is supported.   

This package is a fork of https://github.com/yoavfeld/udger itself forked from https://github.com/udger/udger.

## Tests / Benchmarks

To run tests and benchmarks you need to have the Udger database `udgerdb_v3.dat` located in this folder.

To run unit tests  and run:
```shell
go tests ./...
```

To run tests with benchmarks:
```shell
go test -bench=.
```