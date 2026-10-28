package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	repo := t.TempDir()
	cfg, err := Load(repo, "")
	if err != nil || len(cfg.ExcludeDirs(repo)) != 1 {
		t.Fatalf("missing optional config: %+v, %v", cfg, err)
	}
	path := filepath.Join(repo, FileName)
	if _, err := Load(repo, path); err == nil {
		t.Fatal("explicit missing config must fail")
	}
	for _, tc := range []struct {
		name, yaml string
		wantErr    bool
	}{
		{"valid", "name: demo\nversion: 1.2.3\nexclude: []\n", false},
		{"unknown field", "naem: demo\n", true},
		{"invalid YAML", "name: [\n", true},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(repo, "")
			if (err != nil) != tc.wantErr {
				t.Fatalf("Load: %+v, %v", cfg, err)
			}
			if tc.name == "valid" && (cfg.Name != "demo" || len(cfg.ExcludeDirs(repo)) != 0) {
				t.Fatalf("unexpected config: %+v", cfg)
			}
		})
	}
}
