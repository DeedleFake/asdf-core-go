package shims

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"asdf/config"
	"asdf/internal/versions"
	"asdf/plugins"
	"asdf/repotest"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

const testPluginName = "lua"

func TestGenerateForVersion(t *testing.T) {
	version := "1.1.0"
	version2 := "2.0.0"
	conf, plugin := generateConfig(t)
	installVersion(t, conf, plugin, version)
	installVersion(t, conf, plugin, version2)
	executables, err := ToolExecutables(conf, plugin, version)
	assert.Nil(t, err)

	t.Run("generates shim script for every executable in version", func(t *testing.T) {
		assert.Nil(t, GenerateForVersion(conf, plugin, version))

		// check for generated shims
		for _, executable := range executables {
			shimName := filepath.Base(executable)
			shimPath := Path(conf, shimName)
			assert.Nil(t, unix.Access(shimPath, unix.X_OK))
		}
	})

	t.Run("updates existing shims for every executable in version", func(t *testing.T) {
		assert.Nil(t, GenerateForVersion(conf, plugin, version))
		assert.Nil(t, GenerateForVersion(conf, plugin, version2))

		// check for generated shims
		for _, executable := range executables {
			shimName := filepath.Base(executable)
			shimPath := Path(conf, shimName)
			assert.Nil(t, unix.Access(shimPath, unix.X_OK))
		}
	})
}

func TestWrite(t *testing.T) {
	version := "1.1.0"
	version2 := "2.0.0"
	conf, plugin := generateConfig(t)
	installVersion(t, conf, plugin, version)
	installVersion(t, conf, plugin, version2)
	executables, err := ToolExecutables(conf, plugin, version)
	executable := executables[0]
	assert.Nil(t, err)

	t.Run("writes a new shim file when doesn't exist", func(t *testing.T) {
		executable := executables[0]
		err = Write(conf, plugin, version, executable)
		assert.Nil(t, err)

		// shim is executable
		shimName := filepath.Base(executable)
		shimPath := Path(conf, shimName)
		assert.Nil(t, unix.Access(shimPath, unix.X_OK))

		// shim exists and has expected contents
		content, err := os.ReadFile(shimPath)
		assert.Nil(t, err)
		want := "#!/usr/bin/env bash\n# asdf-plugin: lua 1.1.0\nexec asdf exec \"dummy\" \"$@\""
		assert.Equal(t, want, string(content))
		os.Remove(shimPath)
	})

	t.Run("updates an existing shim file when already present", func(t *testing.T) {
		// Write same shim for two versions
		assert.Nil(t, Write(conf, plugin, version, executable))
		assert.Nil(t, Write(conf, plugin, version2, executable))

		// shim is still executable
		shimName := filepath.Base(executable)
		shimPath := Path(conf, shimName)
		assert.Nil(t, unix.Access(shimPath, unix.X_OK))

		// has expected contents
		content, err := os.ReadFile(shimPath)
		assert.Nil(t, err)
		want := "#!/usr/bin/env bash\n# asdf-plugin: lua 2.0.0\n# asdf-plugin: lua 1.1.0\nexec asdf exec \"dummy\" \"$@\""
		assert.Equal(t, want, string(content))
		os.Remove(shimPath)
	})

	t.Run("doesn't add the same version to a shim file twice", func(t *testing.T) {
		assert.Nil(t, Write(conf, plugin, version, executable))
		assert.Nil(t, Write(conf, plugin, version, executable))

		// Shim doesn't contain any duplicate lines
		shimPath := Path(conf, filepath.Base(executable))
		content, err := os.ReadFile(shimPath)
		assert.Nil(t, err)
		want := "#!/usr/bin/env bash\n# asdf-plugin: lua 1.1.0\nexec asdf exec \"dummy\" \"$@\""
		assert.Equal(t, want, string(content))
		os.Remove(shimPath)
	})
}

func TestToolExecutables(t *testing.T) {
	version := "1.1.0"
	conf, plugin := generateConfig(t)
	installVersion(t, conf, plugin, version)

	t.Run("returns list of executables for plugin", func(t *testing.T) {
		executables, err := ToolExecutables(conf, plugin, version)
		assert.Nil(t, err)

		var filenames []string
		for _, executablePath := range executables {
			assert.True(t, strings.HasPrefix(executablePath, conf.DataDir))
			filenames = append(filenames, filepath.Base(executablePath))
		}

		assert.Equal(t, filenames, []string{"dummy", "other_bin"})
	})
}

func TestExecutableDirs(t *testing.T) {
	conf, plugin := generateConfig(t)
	installVersion(t, conf, plugin, "1.2.3")

	t.Run("returns list only containing 'bin' when list-bin-paths callback missing", func(t *testing.T) {
		executables, err := ExecutableDirs(plugin)
		assert.Nil(t, err)
		assert.Equal(t, executables, []string{"bin"})
	})

	t.Run("returns list of executable paths for tool version", func(t *testing.T) {
		data := []byte("echo 'foo bar'")
		err := os.WriteFile(filepath.Join(plugin.Dir, "bin", "list-bin-paths"), data, 0o777)
		assert.Nil(t, err)

		executables, err := ExecutableDirs(plugin)
		assert.Nil(t, err)
		assert.Equal(t, executables, []string{"foo", "bar"})
	})
}

// Helper functions
func buildOutputs() (strings.Builder, strings.Builder) {
	var stdout strings.Builder
	var stderr strings.Builder

	return stdout, stderr
}

func generateConfig(t *testing.T) (config.Config, plugins.Plugin) {
	t.Helper()
	testDataDir := t.TempDir()
	conf, err := config.LoadConfig()
	assert.Nil(t, err)
	conf.DataDir = testDataDir

	_, err = repotest.InstallPlugin("dummy_plugin", testDataDir, testPluginName)
	assert.Nil(t, err)

	return conf, plugins.New(conf, testPluginName)
}

func installVersion(t *testing.T, conf config.Config, plugin plugins.Plugin, version string) {
	t.Helper()
	stdout, stderr := buildOutputs()
	err := versions.InstallOneVersion(conf, plugin, version, &stdout, &stderr)
	assert.Nil(t, err)
}
