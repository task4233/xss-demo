package main

import (
	"log"

	demo "github.com/task4233/xssdemo"
)

func main() {
	if err := demo.NewServer().Run(); err != nil {
		log.Fatalf("failed Run: %s", err.Error())
	}
}
