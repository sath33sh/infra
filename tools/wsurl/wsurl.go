package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/GeertJohan/go.linenoise"
	"github.com/sath33sh/infra/wapi"
	"github.com/sath33sh/infra/util"
	"os"
	"regexp"
	"strings"
)

type env struct {
	host    string // Host url string.
	credStr string // Credentials string.
	verbose bool   // Enable verbose output.
}

var e env

func vPrintf(format string, v ...interface{}) {
	if e.verbose {
		fmt.Printf(format+"\n", v...)
	}
}

func printRawJson(raw json.RawMessage, err error) {
	var out bytes.Buffer

	json.Indent(&out, raw, "", "  ")
	if err != nil {
		fmt.Print("ERROR: ")
	}
	out.WriteTo(os.Stdout)
	fmt.Println()
}

func newClient(host, credStr string, once bool) (*wapi.Client, error) {
	// Parse credentials string.
	creds := strings.SplitN(credStr, ":", 3)

	return wapi.NewClient(host, creds[0], creds[1], creds[2], once, e.verbose, wapi.NopOnConnError)
}

func exec(c *wapi.Client, rid, method, uri, reqJsonStr string) error {
	var reqData, respData, respErr json.RawMessage

	if len(reqJsonStr) == 0 {
		reqData = json.RawMessage("{}")
	} else {
		reqData = json.RawMessage(reqJsonStr)
	}

	err := c.RestExec(rid, method, uri, &reqData, &respData, &respErr)
	if err != nil {
		if err == util.ErrInternal {
			printRawJson(respErr, err)
		}
	} else {
		printRawJson(respData, nil)
	}

	return err
}

func printShellHelp() {
	fmt.Print(
		"help                Print this help message\n",
		"get <uri> [<data>]  Execute GET method\n",
		"post <uri> [<data>] Execute POST method\n",
		"ping                Ping server\n",
		"clear               Clear screen\n",
		"quit                Quit the shell\n", "\n")
}

func quit(code int) {
	os.Exit(code)
}

func execShell() {
	// Create new client.
	c, err := newClient(e.host, e.credStr, false)
	if err != nil {
		fmt.Printf("Failed to connect to %s: %s\n", e.host, err)
		os.Exit(-2)
	}

	prompt := e.host + "> "
	splitter := regexp.MustCompile(`\s+`)

	for {
		inputline, err := linenoise.Line(prompt)
		if err != nil {
			if err == linenoise.KillSignalError {
				quit(0)
			}
			fmt.Printf("Unexpected error: %s\n", err)
			quit(-1)
		}

		tokens := splitter.Split(inputline, 3)
		if len(tokens) == 0 || len(tokens[0]) == 0 {
			continue
		}

		switch tokens[0] {
		case "help":
			printShellHelp()
		case "get":
			fallthrough
		case "post":
			if len(tokens) < 2 {
				fmt.Printf("Invalid syntax: Type 'help' %d\n", len(tokens))
				continue
			}
			var data string
			if len(tokens) < 3 {
				data = ""
			} else {
				data = tokens[2]
			}
			exec(c, "shell", tokens[0], tokens[1], data)
			linenoise.AddHistory(inputline)
		case "ping":
			exec(c, "shell", "GET", "/ping", "")
		case "clear":
			linenoise.Clear()
		case "quit":
			quit(0)
		default:
			fmt.Printf("Invalid command: Type 'help' %d\n", len(tokens))
		}
	}
}

func execSingleCommand(method, uri, data *string) {
	// Create new client.
	c, err := newClient(e.host, e.credStr, true)
	if err != nil {
		fmt.Printf("Failed to connect to %s: %s\n", e.host, err)
		os.Exit(-2)
	}

	// Execute.
	exec(c, "single", *method, *uri, *data)
}

func parseEnv() {
	e.host = os.Getenv("WSURL_HOST")
	e.credStr = os.Getenv("WSURL_CREDENTIALS")
}

func main() {
	// Parse host and credentials from environment variables.
	parseEnv()

	// Parse command line args.
	cred := flag.String("c", "", "Credentials")
	method := flag.String("m", "", "Method: get, post")
	uri := flag.String("u", "/ping", "URI")
	data := flag.String("d", "", "Data: JSON string")
	flag.BoolVar(&e.verbose, "v", false, "Verbose output")
	help := flag.Bool("h", false, "Print help")
	flag.Parse()

	// Override host & credentials from command line.
	if flag.NArg() > 0 {
		e.host = flag.Arg(0)
	}

	if len(*cred) > 0 {
		e.credStr = *cred
	}

	if *help || len(e.host) == 0 || len(e.credStr) == 0 {
		fmt.Print(
			"Usage: [options...] <host-url>\n",
			"Options:\n",
			" -c CREDENTIALS  <user-id>:<session-id>:<access-token>\n",
			" -m METHOD       Method: get, post, etc\n",
			" -u URI          URI endpoint\n",
			" -d DATA         Data: JSON string\n",
			" -v              Enable verbose output\n",
			" -h              Print this help message\n",
			"\n",
			"Example: wsurl -c 1:ae727ec1:8B730fusiro= -m get -u /ping localhost:8080\n")
		os.Exit(-1)
	}

	// Parse credentials.
	credParams := strings.SplitN(e.credStr, ":", 3)
	if len(credParams) != 3 {
		fmt.Println("Invalid credentials. Expected format: <user-id>:<session-id>:<access-token>")
		os.Exit(-1)
	}

	// Start connection routine.

	if len(*method) == 0 {
		// Execute shell.
		execShell()
	} else {
		// Execute single command.
		execSingleCommand(method, uri, data)
	}
}
