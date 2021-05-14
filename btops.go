package main

import (
	"btops/config"
	"btops/handlers"
	"btops/monitors"
	"bufio"
	"log"
	"os/exec"
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
		monitors, err := monitors.GetMonitors(c.IgnoreMons)
		if err != nil {
			log.Println("Unable to obtain monitors:", err)
		}

		handlers.Handle(monitors)
	}
}
