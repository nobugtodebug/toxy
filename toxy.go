package main

import (
	"toxy"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	config = kingpin.Flag("config", "Config file.").Short('c').Default("toxy.ini").String()
)

func main() {
	kingpin.Parse()
	var toxy = toxy.NewToxy()
	if err := toxy.LoadConfig(*config); err != nil {
		panic(err)
	}
	toxy.Serve()
}