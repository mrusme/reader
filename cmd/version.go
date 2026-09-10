package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

var (
	version = ""
	commit  = ""
	date    = ""
)

const devVersion = "dev"

type buildInfo struct {
	Version  string
	Revision string
	Date     string
}

func currentBuild() buildInfo {
	build := buildInfo{
		Version:  version,
		Revision: commit,
		Date:     date,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		build.fillFrom(info)
	}

	if build.Version == "" {
		build.Version = devVersion
	}
	build.Version = strings.TrimPrefix(build.Version, "v")
	build.Version = strings.TrimSuffix(build.Version, "+dirty")

	return build
}

func (b *buildInfo) fillFrom(info *debug.BuildInfo) {
	if b.Version == "" &&
		info.Main.Version != "" &&
		info.Main.Version != "(devel)" {
		b.Version = info.Main.Version
	}

	injected := b.Revision != ""

	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if b.Revision == "" {
				b.Revision = setting.Value
			}
		case "vcs.time":
			if b.Date == "" {
				b.Date = setting.Value
			}
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if modified && !injected && b.Revision != "" {
		b.Revision += "-dirty"
	}
}

func (b buildInfo) template() string {
	var out strings.Builder

	out.WriteString("{{.DisplayName}} {{.Version}}\n")
	if b.Revision != "" {
		fmt.Fprintf(&out, "revision %s\n", b.Revision)
	}
	if b.Date != "" {
		fmt.Fprintf(&out, "built    %s\n", b.Date)
	}
	fmt.Fprintf(&out, "%s %s/%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	return out.String()
}
