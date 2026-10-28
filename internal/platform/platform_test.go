package platform

import (
	"testing"
	"testing/fstest"
)

func TestLoadPaths(t *testing.T) {
	for _, name := range []string{"../escape", "..", "/absolute", `a\b`, "C:drive", "", "Not-Kebab"} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(fstest.MapFS{"platform.yaml": {Data: []byte("name: '" + name + "'\n")}})
			if err == nil {
				t.Fatalf("accepted unsafe name %q", name)
			}
		})
	}
	for _, dir := range []string{"../escape", "/absolute", `a\..\escape`, "C:/escape"} {
		t.Run(dir, func(t *testing.T) {
			_, err := Load(fstest.MapFS{"platform.yaml": {Data: []byte("name: safe\nskillsDir: '" + dir + "'\n")}})
			if err == nil {
				t.Fatalf("accepted unsafe skillsDir %q", dir)
			}
		})
	}
}

func TestBuiltin(t *testing.T) {
	ps, err := Builtin()
	if err != nil || len(ps) != 2 {
		t.Fatalf("builtin platforms: %v, %v", ps, err)
	}
	if ps[0].Name != "claude-code" || ps[1].Name != "pi" {
		t.Fatalf("unexpected builtin platforms: %s, %s", ps[0].Name, ps[1].Name)
	}
	if _, err := Lookup("gemini-cli"); err == nil {
		t.Fatal("Gemini should no longer be a builtin platform")
	}
	if _, err := Lookup("missing"); err == nil {
		t.Fatal("expected unknown platform error")
	}
}
