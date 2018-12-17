# apache-log

An apache log parser for analyzing log files. Currently tailored to the log
format:

```apache
LogFormat "%h %l %u %t \"%r\" %>s %O \"%{Referer}i\" \"%{User-Agent}i\""
```

For more info on apache log format, see
[apache log documentation](http://httpd.apache.org/docs/current/mod/mod_log_config.html)

## Usage

```sh
Apache Log Utils

    apache-log [flags] ...[input]

Usage of apache-log:
  -max int
    	filters entries exceeding max occurrence
  -min int
    	filters entries not meeting min occurrence
  -status int
    	filter by status code (default 200)
  -v	verbose log output
  -x string
    	exclude URLs starting with string (comma separated for multiple)
```
