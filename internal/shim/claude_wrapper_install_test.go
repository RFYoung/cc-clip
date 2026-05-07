package shim

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setupFakeHome creates a t.TempDir() with .local/bin/ subdir and returns
// the bin dir path. Tests use this as a fake remote $HOME root.
func setupFakeHome(t *testing.T) (home, binDir string) {
	t.Helper()
	home = t.TempDir()
	binDir = filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("setup binDir: %v", err)
	}
	return home, binDir
}

func TestClassifyClaudeBin_None(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash not reliably available on Windows runner")
	}
	home, _ := setupFakeHome(t)
	s := &localSession{home: home}
	kind, err := classifyClaudeBin(s)
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if kind != "none" {
		t.Fatalf("got %q, want none", kind)
	}
}

func TestClassifyClaudeBin_RegularNonWrapper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash not reliably available on Windows runner")
	}
	home, binDir := setupFakeHome(t)
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte("\x7fELF... fake"), 0755); err != nil {
		t.Fatal(err)
	}
	s := &localSession{home: home}
	kind, err := classifyClaudeBin(s)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "regular" {
		t.Fatalf("got %q, want regular", kind)
	}
}

func TestClassifyClaudeBin_CcWrapper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash not reliably available on Windows runner")
	}
	home, binDir := setupFakeHome(t)
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte(ClaudeWrapperScript(18339)), 0755); err != nil {
		t.Fatal(err)
	}
	s := &localSession{home: home}
	kind, err := classifyClaudeBin(s)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "cc_wrapper" {
		t.Fatalf("got %q, want cc_wrapper", kind)
	}
}

func TestClassifyClaudeBin_Symlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	home, binDir := setupFakeHome(t)
	target := filepath.Join(home, ".local", "share", "claude", "versions", "2.1.132")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("\x7fELF... real binary content"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(binDir, "claude")); err != nil {
		t.Fatal(err)
	}
	s := &localSession{home: home}
	kind, err := classifyClaudeBin(s)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "symlink" {
		t.Fatalf("got %q, want symlink", kind)
	}
}
