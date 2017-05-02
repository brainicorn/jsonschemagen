package main

import (
	"github.com/brainicorn/jsonschemagen/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {

	root := cmd.NewRootCommand()
	doc.GenMarkdownTree(root.Cmd, "./")
}
