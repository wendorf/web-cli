package main

import (
	"github.com/cloudfoundry/cli/maincli"
	"github.com/gopherjs/gopherjs/js"
)

func main() {
	js.Global.Set("cf", callCommand)
}

func callCommand(args ...string) {
	go func() {
		maincli.Maincli(append([]string{"./web-cli"}, args...))
	}()
}
