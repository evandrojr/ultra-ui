// Command ultra-ui e o binario do plugin de abas GUI do herdr.
package main

import (
	"os"

	"ultra-ui/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
