package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIMutex_AutoRecoverWithTokenOnly(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "connect", "fakehost.invalid", "--auto-recover", "--token-only")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit on flag conflict")
	}
	if !strings.Contains(stderr.String(), "--auto-recover cannot be combined with --token-only") {
		t.Fatalf("missing mutex error in stderr: %s", stderr.String())
	}
}
