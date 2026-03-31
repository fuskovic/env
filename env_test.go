package env

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tmpEnv(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBasicTypes(t *testing.T) {
	path := tmpEnv(t, `
HOST=localhost
PORT=8080
RATE=1.5
DEBUG=true
TIMEOUT=5s
`)

	var cfg struct {
		Host    string        `env:"HOST"`
		Port    int           `env:"PORT"`
		Rate    float64       `env:"RATE"`
		Debug   bool          `env:"DEBUG"`
		Timeout time.Duration `env:"TIMEOUT"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
	if cfg.Rate != 1.5 {
		t.Errorf("Rate = %f, want %f", cfg.Rate, 1.5)
	}
	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 5*time.Second)
	}
}

func TestSlice(t *testing.T) {
	path := tmpEnv(t, `HOSTS=a,b,c`)

	var cfg struct {
		Hosts []string `env:"HOSTS"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Hosts) != 3 || cfg.Hosts[0] != "a" || cfg.Hosts[1] != "b" || cfg.Hosts[2] != "c" {
		t.Errorf("Hosts = %v, want [a b c]", cfg.Hosts)
	}
}

func TestIntSlice(t *testing.T) {
	path := tmpEnv(t, `PORTS=80, 443, 8080`)

	var cfg struct {
		Ports []int `env:"PORTS"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Ports) != 3 || cfg.Ports[0] != 80 || cfg.Ports[1] != 443 || cfg.Ports[2] != 8080 {
		t.Errorf("Ports = %v, want [80 443 8080]", cfg.Ports)
	}
}

func TestRequired(t *testing.T) {
	path := tmpEnv(t, `FOO=bar`)

	var cfg struct {
		Missing string `env:"MISSING,required"`
	}

	err := UnmarshalFromFile(path, &cfg)
	if err == nil {
		t.Fatal("expected error for missing required key")
	}
}

func TestDefault(t *testing.T) {
	path := tmpEnv(t, `FOO=bar`)

	var cfg struct {
		Port int `env:"PORT,default=3000"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 3000)
	}
}

func TestQuotedValues(t *testing.T) {
	path := tmpEnv(t, `
DOUBLE="hello world"
SINGLE='foo bar'
`)

	var cfg struct {
		Double string `env:"DOUBLE"`
		Single string `env:"SINGLE"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Double != "hello world" {
		t.Errorf("Double = %q, want %q", cfg.Double, "hello world")
	}
	if cfg.Single != "foo bar" {
		t.Errorf("Single = %q, want %q", cfg.Single, "foo bar")
	}
}

func TestCommentsAndExport(t *testing.T) {
	path := tmpEnv(t, `
# this is a comment
export KEY=value
`)

	var cfg struct {
		Key string `env:"KEY"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Key != "value" {
		t.Errorf("Key = %q, want %q", cfg.Key, "value")
	}
}

func TestNestedStruct(t *testing.T) {
	path := tmpEnv(t, `
DB_HOST=localhost
DB_PORT=5432
APP_NAME=myapp
`)

	type DB struct {
		Host string `env:"DB_HOST"`
		Port int    `env:"DB_PORT"`
	}

	var cfg struct {
		DB
		AppName string `env:"APP_NAME"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "localhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 5432)
	}
	if cfg.AppName != "myapp" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "myapp")
	}
}

func TestPointer(t *testing.T) {
	path := tmpEnv(t, `VAL=42`)

	var cfg struct {
		Val *int `env:"VAL"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Val == nil || *cfg.Val != 42 {
		t.Errorf("Val = %v, want ptr to 42", cfg.Val)
	}
}

func TestUntaggedFieldsIgnored(t *testing.T) {
	path := tmpEnv(t, `FOO=bar`)

	var cfg struct {
		Foo     string `env:"FOO"`
		Ignored string
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Foo != "bar" {
		t.Errorf("Foo = %q, want %q", cfg.Foo, "bar")
	}
	if cfg.Ignored != "" {
		t.Errorf("Ignored = %q, want empty", cfg.Ignored)
	}
}

func TestNonPointerError(t *testing.T) {
	var cfg struct{}
	err := Unmarshal(cfg)
	if err == nil {
		t.Fatal("expected error for non-pointer dst")
	}
}

func TestFileNotFound(t *testing.T) {
	var cfg struct{}
	err := UnmarshalFromFile("/nonexistent/.env", &cfg)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestUnsignedInt(t *testing.T) {
	path := tmpEnv(t, `COUNT=255`)

	var cfg struct {
		Count uint8 `env:"COUNT"`
	}

	if err := UnmarshalFromFile(path, &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Count != 255 {
		t.Errorf("Count = %d, want 255", cfg.Count)
	}
}

func TestUnmarshal(t *testing.T) {
	t.Setenv("TEST_ENV_HOST", "127.0.0.1")
	t.Setenv("TEST_ENV_PORT", "9090")
	t.Setenv("TEST_ENV_DEBUG", "true")

	var cfg struct {
		Host  string `env:"TEST_ENV_HOST"`
		Port  int    `env:"TEST_ENV_PORT"`
		Debug bool   `env:"TEST_ENV_DEBUG"`
	}

	if err := Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", cfg.Host, "127.0.0.1")
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9090)
	}
	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
}

func TestUnmarshalRequired(t *testing.T) {
	var cfg struct {
		Missing string `env:"TEST_ENV_DEFINITELY_NOT_SET,required"`
	}

	err := Unmarshal(&cfg)
	if err == nil {
		t.Fatal("expected error for missing required env var")
	}
}

func TestUnmarshalDefault(t *testing.T) {
	var cfg struct {
		Port int `env:"TEST_ENV_UNSET_PORT,default=3000"`
	}

	if err := Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 3000)
	}
}
