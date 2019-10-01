package main

import (
	"bufio"
	"log"
	"os/exec"

	"github.com/roberteinhaus/btops/config"
	"github.com/roberteinhaus/btops/handlers"
	"github.com/roberteinhaus/btops/monitors"
)

func main() {
	for {
		listen()
	}
}

func listen() {
	c, err := config.GetConfig()
	if err != nil {
		log.Fatal("Unable to get config", err)
	}

	log.Println("Config: ", c)

	handlers := handlers.NewHandlers(c)

	bspc_sub := exec.Command("bspc", "subscribe", "report")
	bspcReader, err := bspc_sub.StdoutPipe()

	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(bspcReader)

	err = bspc_sub.Start()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err = bspc_sub.Wait()
		if err != nil {
			log.Fatal(err)
		}
	}()

	for !c.ConfigChanged() && scanner.Scan() {
		monitors, err := monitors.GetMonitors()
		if err != nil {
			log.Println("Unable to obtain monitors:", err)
		}

		handlers.Handle(monitors)
	}
}
