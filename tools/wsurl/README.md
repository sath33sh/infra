Websocket URL (wsurl) is a command line tool for interacting with Web API Service (WAPI) via websocket. It has an interface similar to curl.
<code><pre>
$ wsurl -h
Usage: [options...] \<host-url\>
Options:
 -c CREDENTIALS  \<user-id\>:\<session-id\>:\<access-token\>
 -m METHOD       Method: get, post, etc
 -u URI          URI endpoint
 -d DATA         Data: JSON string
 -v              Enable verbose output
 -h              Print this help message
</pre></code>

### Single command mode
Specifying the "-m" option while invoking wsurl executes the command once.
<code><pre>
$ wsurl -c 1:ae727ec1:8B730fusiro= -m get -u /v1.0/channel/show/5 107.178.223.208
{
  "type": "channel",
  "id": 5,
  "name": "DB test",
  "description": "Database test",
  "menu": [
    {
      "type": "topics",
      "title": "Topics"
    }
  ],
  "options": {
    "discover": {
      "geo": {
        "type": "",
        "coordinates": [
          0,
          0
        ]
      }
    },
    "notify": {}
  },
  "createdAt": "2015-06-14T00:35:48-07:00"
}
</pre></code>

### Shell mode
Shell mode is convenient for issuing a series of commands. It is invoked by not specifying the "-m" option.
<code><pre>
$ wsurl -c 1:ae727ec1:8B730fusiro= 107.178.223.208
localhost:8080> help
help                Print this help
get \<uri\> [\<data\>]  Execute GET method
post \<uri\> [\<data\>] Execute POST method
ping                Ping server
clear               Clear screen
quit                Quit the shell
localhost:8080> 
localhost:8080> get /v1.0/channel/show/5
{
  "type": "channel",
  "id": 5,
  "name": "DB test",
  "description": "Database test",
  "menu": [
    {
      "type": "topics",
      "title": "Topics"
    }
  ],
  "options": {
    "discover": {
      "geo": {
        "type": "",
        "coordinates": [
          0,
          0
        ]
      }
    },
    "notify": {}
  },
  "createdAt": "2015-06-14T00:35:48-07:00"
}
localhost:8080>
</pre></code>

### Environment variables
Credentials and host can be set as environment variables if you don't want to enter them every time.
<code><pre>
$ export WSURL_HOST=107.178.223.208
$ export WSURL_CREDENTIALS=1:ae727ec1:8B730fusiro=
$ wsurl 
107.178.223.208> 
</pre></code>

By default, wsurl connects to the server in secure (TLS) mode. TLS can be disabled using the following environment variable.
<code><pre>
$ export WAPI_SECURE=false
$ wsurl
</pre></code>

### Compile
If you have Go installed, you can compile the latest version of wsurl:
<code><pre>
go get -u github.com/sath33sh/infra/tools/wsurl
</pre></code>
