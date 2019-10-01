package ipc

import (
	"bytes"
	"os/exec"
)

func Send(cmd ...string) (response []byte, err error) {
	bspc := exec.Command("bspc", cmd...)

	var bspcresponse bytes.Buffer
	bspc.Stdout = &bspcresponse

	bspc.Run()

	return bspcresponse.Bytes(), nil
}
