# refurbed-cmd-line-tool

refurbed-cmd-line-tool is a tool to help send http POST messages to a configured url at a configured rate.

## usage: 
```
notify --url=URL [<flags>]
```

Flags:
--help Show context-sensitive help (also
try --help-long and --help-man).

-i, --interval=5s Notification interval

## Sample 
```
go run main.go notify --url=http://localhost:8080/notify --interval=5s < messages.txt
```
