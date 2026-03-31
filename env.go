// Package env unmarshals .env files into Go structs.
package env

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Unmarshal populates dst from the current process environment.
// dst must be a non-nil pointer to a struct. Struct fields are matched to
// environment variable names using the "env" struct tag. Supported tag options:
//
//	type Config struct {
//	    Port    int           `env:"PORT"`
//	    Debug   bool          `env:"DEBUG"`
//	    Timeout time.Duration `env:"TIMEOUT"`
//	    Hosts   []string      `env:"HOSTS"`     // comma-separated
//	}
//
// A "required" option causes Unmarshal to return an error if the key is absent:
//
//	Host string `env:"HOST,required"`
//
// A "default" option provides a fallback value:
//
//	Port int `env:"PORT,default=8080"`
func Unmarshal(dst any) error {
	vals := make(map[string]string)
	for _, entry := range os.Environ() {
		if k, v, ok := strings.Cut(entry, "="); ok {
			vals[k] = v
		}
	}
	return decode(vals, dst)
}

// UnmarshalFromFile reads the file at path and populates dst with the values found.
// It follows the same struct tag conventions as [Unmarshal].
func UnmarshalFromFile(path string, dst any) error {
	vals, err := parse(path)
	if err != nil {
		return fmt.Errorf("env: parsing %s: %w", path, err)
	}
	return decode(vals, dst)
}

// parse reads a .env file and returns a map of key-value pairs.
func parse(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vals := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip blank lines and comments
		if line == "" || line[0] == '#' {
			continue
		}

		// strip optional "export " prefix
		line = strings.TrimPrefix(line, "export ")

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		// strip matching quotes
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}

		vals[key] = val
	}
	return vals, scanner.Err()
}

// decode populates dst from the key-value map.
func decode(vals map[string]string, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("env: dst must be a non-nil pointer to a struct")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("env: dst must be a pointer to a struct, got pointer to %s", rv.Kind())
	}
	return decodeStruct(vals, rv)
}

func decodeStruct(vals map[string]string, rv reflect.Value) error {
	rt := rv.Type()
	for i := range rt.NumField() {
		field := rt.Field(i)
		fv := rv.Field(i)

		if !fv.CanSet() {
			continue
		}

		// Handle embedded/nested structs without an env tag.
		if field.Type.Kind() == reflect.Struct && field.Tag.Get("env") == "" {
			if field.Type == reflect.TypeOf(time.Duration(0)) {
				continue // duration is a struct-like type handled below
			}
			if err := decodeStruct(vals, fv); err != nil {
				return err
			}
			continue
		}

		tag := field.Tag.Get("env")
		if tag == "" || tag == "-" {
			continue
		}

		name, opts := parseTag(tag)
		raw, ok := vals[name]
		if !ok {
			if def, hasDef := opts["default"]; hasDef {
				raw = def
			} else if _, req := opts["required"]; req {
				return fmt.Errorf("env: required key %q not set", name)
			} else {
				continue
			}
		}

		if err := setField(fv, raw); err != nil {
			return fmt.Errorf("env: setting %s (%s): %w", name, field.Type, err)
		}
	}
	return nil
}

// parseTag splits "KEY,required,default=val" into the key name and options map.
func parseTag(tag string) (string, map[string]string) {
	parts := strings.Split(tag, ",")
	name := parts[0]
	opts := make(map[string]string, len(parts)-1)
	for _, p := range parts[1:] {
		k, v, _ := strings.Cut(p, "=")
		opts[k] = v
	}
	return name, opts
}

// setField assigns the string value to the reflect.Value, handling type conversion.
func setField(fv reflect.Value, raw string) error {
	// Handle pointer types.
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return setField(fv.Elem(), raw)
	}

	// Handle slices (except []byte).
	if fv.Kind() == reflect.Slice && fv.Type().Elem().Kind() != reflect.Uint8 {
		return setSlice(fv, raw)
	}

	// time.Duration
	if fv.Type() == reflect.TypeFor[time.Duration]() {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}
		fv.Set(reflect.ValueOf(d))
		return nil
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetFloat(f)
	default:
		return fmt.Errorf("unsupported type %s", fv.Type())
	}
	return nil
}

// setSlice splits raw on commas and assigns each element to the slice.
func setSlice(fv reflect.Value, raw string) error {
	parts := strings.Split(raw, ",")
	slice := reflect.MakeSlice(fv.Type(), len(parts), len(parts))
	for i, p := range parts {
		if err := setField(slice.Index(i), strings.TrimSpace(p)); err != nil {
			return err
		}
	}
	fv.Set(slice)
	return nil
}
