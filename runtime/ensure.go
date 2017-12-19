package runtime

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Southclaws/sampctl/download"
	"github.com/Southclaws/sampctl/types"
	"github.com/Southclaws/sampctl/util"
	"github.com/pkg/errors"
)

// Ensure will make sure a Config's dir is representative of the held configuration.
// If any of the following are missing or mismatching, they will be automatically downloaded:
// - Server binaries (server, announce, npc)
// - Plugin binaries
// and a `server.cfg` is generated based on the contents of the Config fields.
func Ensure(cfg *types.Runtime) (err error) {
	cacheDir, err := download.GetCacheDir()
	if err != nil {
		return err
	}

	err = EnsureBinaries(*cfg)
	if err != nil {
		return
	}

	err = EnsurePlugins(cfg, cacheDir)
	if err != nil {
		return
	}

	err = EnsureScripts(*cfg)
	if err != nil {
		return
	}

	err = GenerateServerCfg(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to generate server.cfg")
	}

	return
}

// EnsureBinaries ensures the dir has all the necessary files to run a server
func EnsureBinaries(cfg types.Runtime) (err error) {
	missing := false

	if !util.Exists(filepath.Join(cfg.WorkingDir, getNpcBinary(cfg.Platform))) {
		missing = true
	}
	if !util.Exists(filepath.Join(cfg.WorkingDir, getAnnounceBinary(cfg.Platform))) {
		missing = true
	}
	if !util.Exists(filepath.Join(cfg.WorkingDir, getServerBinary(cfg.Platform))) {
		missing = true
	}

	if missing {
		err = GetServerPackage(*cfg.Endpoint, *cfg.Version, cfg.WorkingDir, cfg.Platform)
		if err != nil {
			return errors.Wrap(err, "failed to get runtime package")
		}
	}

	ok, err := MatchesChecksum(filepath.Join(cfg.WorkingDir, getServerBinary(cfg.Platform)), cfg.Platform, *cfg.Version)
	if err != nil {
		return errors.Wrap(err, "failed to match checksum")
	} else if !ok {
		return errors.Errorf("existing binary does not match checksum for version %s", *cfg.Version)
	}

	return
}

// EnsureScripts checks that all the declared scripts are present
func EnsureScripts(cfg types.Runtime) (err error) {
	errs := []string{}

	for _, gamemode := range cfg.Gamemodes {
		fullpath := filepath.Join(cfg.WorkingDir, "gamemodes", gamemode+".amx")
		if !util.Exists(fullpath) {
			errs = append(errs, fmt.Sprintf("gamemode '%s' is missing its .amx file from the gamemodes directory", gamemode))
		}
	}
	for _, filterscript := range cfg.Filterscripts {
		fullpath := filepath.Join(cfg.WorkingDir, "filterscripts", filterscript+".amx")
		if !util.Exists(fullpath) {
			errs = append(errs, fmt.Sprintf("filterscript '%s' is missing its .amx file from the filterscripts directory", filterscript))
		}
	}

	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, ", "))
	}

	return
}

func pluginExtensionForOS(os string) (ext string) {
	switch os {
	case "windows":
		ext = ".dll"
	case "linux", "darwin":
		ext = ".so"
	}
	return
}
