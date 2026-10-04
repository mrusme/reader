package cmd

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestCurrentBuildFromLDFlags(t *testing.T) {
	restore := func(v, c, d string) { version, commit, date = v, c, d }
	defer restore(version, commit, date)

	version, commit, date = "v1.2.3", "abcdef", "2026-01-02T03:04:05Z"

	build := currentBuild()

	if build.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", build.Version, "1.2.3")
	}
	if build.Revision != "abcdef" {
		t.Errorf("revision = %q, want %q", build.Revision, "abcdef")
	}
	if build.Date != "2026-01-02T03:04:05Z" {
		t.Errorf("date = %q, want %q", build.Date, "2026-01-02T03:04:05Z")
	}
}

func TestCurrentBuildFallsBackToDev(t *testing.T) {
	restore := func(v, c, d string) { version, commit, date = v, c, d }
	defer restore(version, commit, date)

	version, commit, date = "", "", ""

	if build := currentBuild(); build.Version == "" {
		t.Error("version is empty")
	}
}

func TestFillFrom(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.9.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "cafebabe"},
			{Key: "vcs.time", Value: "2026-02-03T04:05:06Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	var build buildInfo
	build.fillFrom(info)

	if build.Version != "v0.9.0" {
		t.Errorf("version = %q, want %q", build.Version, "v0.9.0")
	}
	if build.Revision != "cafebabe-dirty" {
		t.Errorf("revision = %q, want %q", build.Revision, "cafebabe-dirty")
	}
	if build.Date != "2026-02-03T04:05:06Z" {
		t.Errorf("date = %q, want %q", build.Date, "2026-02-03T04:05:06Z")
	}
}

func TestFillFromKeepsInjectedValues(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.9.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "cafebabe"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	build := buildInfo{Version: "1.0.0", Revision: "deadbeef"}
	build.fillFrom(info)

	if build.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", build.Version, "1.0.0")
	}
	if build.Revision != "deadbeef" {
		t.Errorf("revision = %q, want %q", build.Revision, "deadbeef")
	}
}

func TestCurrentBuildDropsDirtyVersionSuffix(t *testing.T) {
	restore := func(v, c, d string) { version, commit, date = v, c, d }
	defer restore(version, commit, date)

	version, commit, date = "v1.2.3+dirty", "", ""

	if build := currentBuild(); build.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", build.Version, "1.2.3")
	}
}

func TestFillFromIgnoresDevelVersion(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}

	var build buildInfo
	build.fillFrom(info)

	if build.Version != "" {
		t.Errorf("version = %q, want empty", build.Version)
	}
}

func TestTemplate(t *testing.T) {
	build := buildInfo{
		Version:  "1.2.3",
		Revision: "abcdef",
		Date:     "2026-01-02T03:04:05Z",
	}

	got := build.template()

	for _, want := range []string{
		"{{.DisplayName}} {{.Version}}\n", "abcdef", "2026-01-02T03:04:05Z",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("template does not contain %q:\n%s", want, got)
		}
	}
}

func TestTemplateWithoutVCSInfo(t *testing.T) {
	got := buildInfo{Version: "1.2.3"}.template()

	if !strings.HasPrefix(got, "{{.DisplayName}} {{.Version}}\n") {
		t.Errorf("template does not start with the version line:\n%s", got)
	}
	if strings.Contains(got, "revision") {
		t.Errorf("template mentions a revision it does not have:\n%s", got)
	}
}
