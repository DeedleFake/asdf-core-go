// Package installs contains tool installation logic. It is "dumb" when it comes
// to versions and treats versions as opaque strings. It cannot depend on the
// versions package because the versions package relies on this page.
package installs

import (
	"io/fs"
	"os"
	"path/filepath"

	"asdf/internal/config"
	"asdf/internal/data"
	"asdf/internal/plugins"
	"asdf/internal/toolversions"
)

// Installed returns a slice of all installed versions for a given plugin
func Installed(conf config.Config, plugin plugins.Plugin) (versions []string, err error) {
	installDirectory := data.InstallDirectory(conf.DataDir, plugin.Name)
	files, err := os.ReadDir(installDirectory)
	if err != nil {
		if _, ok := err.(*fs.PathError); ok {
			return versions, nil
		}

		return versions, err
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		versions = append(versions, file.Name())
	}

	return versions, err
}

// InstallPath returns the path to a tool installation
func InstallPath(conf config.Config, plugin plugins.Plugin, versionType, version string) string {
	if versionType == "path" {
		return version
	}

	return filepath.Join(data.InstallDirectory(conf.DataDir, plugin.Name), toolversions.FormatForFS(versionType, version))
}

// DownloadPath returns the download path for a particular plugin and version
func DownloadPath(conf config.Config, plugin plugins.Plugin, versionType, version string) string {
	if versionType == "path" {
		return ""
	}

	return filepath.Join(data.DownloadDirectory(conf.DataDir, plugin.Name), toolversions.FormatForFS(versionType, version))
}

// IsInstalled checks if a specific version of a tool is installed
func IsInstalled(conf config.Config, plugin plugins.Plugin, versionType, version string) bool {
	installDir := InstallPath(conf, plugin, versionType, version)

	// Check if version already installed
	_, err := os.Stat(installDir)
	return !os.IsNotExist(err)
}
