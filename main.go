// Simple go http server for yum offline repo.
package main

import (
	"os"
	"github.com/leocd/yum-go/server"
)

func main() {
	os.Exit(server.Main())
}
