package shim

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestInstall_RegularFileOrigin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash-based install path is Linux-only")
	}
	home, binDir := setupFakeHome(t)
	originalContent := []byte("\x7fELF... pretend this is the real 250MB claude binary")
	if err := os.WriteFile(filepath.Join(binDir, "claude"), originalContent, 0755); err != nil {
		t.Fatal(err)
	}
	s := &localSession{home: home}

	if err := InstallRemoteClaudeWrapper(s, 18339); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Sidecar must hold the original (verbatim).
	sidecar, err := os.ReadFile(filepath.Join(binDir, "claude.cc-clip-real"))
	if err != nil {
		t.Fatalf("sidecar missing: %v", err)
	}
	if string(sidecar) != string(originalContent) {
		t.Fatal("sidecar does not contain original content (mv may have leaked or content was rewritten)")
	}

	// claude must now be the wrapper.
	data, err := os.ReadFile(filepath.Join(binDir, "claude"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# cc-clip claude wrapper") {
		t.Fatal("claude is not the cc-clip wrapper after install")
	}
	// Port-substitution assertion (per T6 review note): wrapper must reference
	// the port we passed to InstallRemoteClaudeWrapper.
	if !strings.Contains(string(data), "18339") {
		t.Fatal("installed wrapper does not contain expected port 18339")
	}

	// claude must be a regular file (not a symlink).
	info, err := os.Lstat(filepath.Join(binDir, "claude"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("claude should be a regular file after install")
	}
}

func TestInstall_NoPriorInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash-based install path is Linux-only")
	}
	home, binDir := setupFakeHome(t)
	s := &localSession{home: home}

	if err := InstallRemoteClaudeWrapper(s, 18339); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Wrapper should now exist as a regular file at ~/.local/bin/claude.
	info, err := os.Lstat(filepath.Join(binDir, "claude"))
	if err != nil {
		t.Fatalf("claude not installed: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("claude should be a regular file, got symlink")
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Fatal("claude should be executable")
	}

	// No sidecar should have been created (no origin to displace).
	if _, err := os.Lstat(filepath.Join(binDir, "claude.cc-clip-real")); !os.IsNotExist(err) {
		t.Fatalf("sidecar should not exist on first install of 'none' case, got: %v", err)
	}

	// Content must be the wrapper script.
	data, err := os.ReadFile(filepath.Join(binDir, "claude"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# cc-clip claude wrapper") {
		t.Fatal("installed file is not the cc-clip wrapper")
	}
}
