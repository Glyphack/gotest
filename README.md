# gotest

Better go test

## Why?

`go test` output is very verbose, for large projects it's hard to find what tests failed, and what was their output. You need to scroll back, use `grep` and other tricks to find this crucial information.

This project offers better experience in the terminal for running tests.

- Summary of tests in the package
- Skips packages without any test
- Prints Details of failed tests at the end


## Quick start

```
go install github.com/glyphack/gotest
```

Then run `gotest` in a go project.
