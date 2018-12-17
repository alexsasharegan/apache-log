# apache-log

An apache log parser for analyzing log files. Currently tailored to the log
format:

```apache
LogFormat "%h %l %u %t \"%r\" %>s %O \"%{Referer}i\" \"%{User-Agent}i\""
```

## Usage

```sh
Apache Log Utils

    apache-log [flags] ...[input]

Usage of apache-log:
  -max int
        filters entries by a maximum occurrence
  -min int
        filters entries by a minimum occurrence
  -status int
        filter by status code (default 200)
  -v    verbose log output
```
