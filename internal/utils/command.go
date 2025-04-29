package utils

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Openbrowser(url string) error {
	if url == "" {
		return fmt.Errorf("URL не может быть пустым")
	}
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("cmd", "/c", "start", url).Start() // Альтернатива rundll32
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}
