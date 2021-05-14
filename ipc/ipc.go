package ipc

import (
	"bytes"
	"log"
	"os/exec"
)

func Send(cmd ...string) (response []byte, err error) {
	bspc := exec.Command("bspc", cmd...)
	log.Printf("send %+v\n", cmd)

	var bspcresponse bytes.Buffer
	bspc.Stdout = &bspcresponse

	bspc.Run()

	return bspcresponse.Bytes(), nil
}
