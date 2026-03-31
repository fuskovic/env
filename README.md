# env

Unmarshal `.env` files directly into Go structs. Zero external dependencies.

## Why?

| Package | Reads `.env` files | Unmarshals to structs | Zero deps |
|---|---|---|---|
| [godotenv](https://github.com/joho/godotenv) | Yes | No | Yes |
| [envconfig](https://github.com/kelseyhightower/envconfig) | No | Yes | Yes |
| [viper](https://github.com/spf13/viper) | Yes | Yes | No (17+ deps) |
| **env** | **Yes** | **Yes** | **Yes** |

**godotenv** loads `.env` files into `os.Environ` but gives you a `map[string]string` — you still have to manually pluck values out and convert types yourself.

**envconfig** maps environment variables to struct fields with type conversion, but it reads from `os.Environ` only — it can't read `.env` files. The common workaround is chaining `godotenv.Load()` then `envconfig.Process()`, which pollutes the process environment and requires two packages for one job.

**viper** can do both, but it's a 17+ dependency kitchen-sink config framework. If all you need is `.env` -> struct, that's a lot of weight.

**env** does one thing: read a `.env` file and unmarshal it into a struct. One function call, one dependency, no environment pollution.

## Install

```
go get github.com/fuskovic/env
```

## Usage

Given a `.env` file:

```
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp

# App
DEBUG=true
TIMEOUT=30s
ALLOWED_ORIGINS=https://example.com, https://api.example.com
```

Unmarshal it into a struct:

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fuskovic/env"
)

type DB struct {
	Host string `env:"DB_HOST"`
	Port int    `env:"DB_PORT"`
	Name string `env:"DB_NAME"`
}

type Config struct {
	DB
	Debug          bool          `env:"DEBUG"`
	Timeout        time.Duration `env:"TIMEOUT"`
	AllowedOrigins []string      `env:"ALLOWED_ORIGINS"`
}

func main() {
	var cfg Config
	if err := env.Unmarshal(".env", &cfg); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", cfg)
	// {DB:{Host:localhost Port:5432 Name:myapp} Debug:true Timeout:30s AllowedOrigins:[https://example.com https://api.example.com]}
}
```

## Struct tags

The `env` struct tag maps a field to a key in the `.env` file.

```go
type Config struct {
	// Basic mapping
	Host string `env:"HOST"`

	// Required — returns an error if the key is missing
	Secret string `env:"SECRET,required"`

	// Default — used when the key is missing
	Port int `env:"PORT,default=8080"`

	// Ignored
	Internal string `env:"-"`
}
```

## Supported types

- `string`
- `bool`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`
- `time.Duration`
- Pointers to any of the above
- Slices of any of the above (comma-separated)
- Embedded structs (fields are flattened)

## `.env` file format

```
# Comments are ignored
KEY=value

# Quotes are stripped
GREETING="hello world"
SINGLE='foo bar'

# export prefix is stripped
export API_KEY=secret
```

## License

MIT
