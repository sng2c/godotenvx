package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/joho/godotenv"
)

type EnvxItem struct {
	Key    string
	Value  string
	Locked bool
}
type EnvxMap map[string]*EnvxItem

func NewEnvxItemFromLine(line string, locked bool) (*EnvxItem, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) == 2 {
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if len(k) == 0 {
			return nil, errors.New("invalid key")
		}
		return &EnvxItem{
			Key:    k,
			Value:  v,
			Locked: locked,
		}, nil
	}
	return nil, errors.New("invalid line")
}

func IsFullComment(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "#")
}

func parsingPlan(envFilePath string) ([]string, error) {
	// pkg1.pkg2.pkg3.env 와 같은 파일 형식이면, 상위패키지부터 나열
	// 아니면 그 파일만 나열
	if envFilePath == "" {
		return nil, fmt.Errorf("empty file path")
	}

	// Handle absolute and relative paths
	dir := filepath.Dir(envFilePath)
	base := filepath.Base(envFilePath)
	parts := strings.Split(base, ".")
	if len(parts) > 2 {
		var result []string
		pkgBegan := false
		for i := 0; i < len(parts); i++ {
			if parts[i] == "env" {
				pkgBegan = true
			}
			if pkgBegan {
				fileName := strings.Join(parts[:i+1], ".")
				result = append(result, filepath.Join(dir, fileName))
			}
		}
		return result, nil
	}

	return []string{envFilePath}, nil
}

func copyEnvLines() []string {
	environ := os.Environ()
	result := make([]string, len(environ))
	copy(result, environ)
	return result
}

// 풀라인 주석으로 # LOCK 바로 다음 ITEM은 locked=true가 된다.
func parseEnvLines(envLines []string) EnvxMap {
	result := make(EnvxMap)

	locked := false
	for _, envLine := range envLines {
		if IsFullComment(envLine) {
			if strings.HasPrefix(envLine, "# LOCK") {
				locked = true
			}
			continue
		}
		envxItem, err := NewEnvxItemFromLine(envLine, locked)
		if err != nil {
			continue
		}
		result[envxItem.Key] = envxItem
		locked = false
	}
	return result
}

func diffEnvxMaps(beforeMap, afterMap EnvxMap) [][3]string {
	var result [][3]string
	seen := make(map[string]bool)

	// left direction
	for key, value := range afterMap {
		beforeValue := ""
		if beforeMap[key] != nil {
			beforeValue = beforeMap[key].Value
		}

		if beforeValue != value.Value {
			diff := [3]string{key, beforeValue, value.Value}
			seenKey := strings.Join([]string{diff[0], diff[1], diff[2]}, ":")
			if seen[seenKey] {
				continue
			}
			result = append(result, diff)
			seen[seenKey] = true
		}
	}
	// right direction
	for key, value := range beforeMap {
		afterValue := ""
		if afterMap[key] != nil {
			afterValue = afterMap[key].Value
		}
		if afterValue != value.Value {
			diff := [3]string{key, value.Value, afterValue}
			seenKey := strings.Join([]string{diff[0], diff[1], diff[2]}, ":")
			if seen[seenKey] {
				continue
			}
			result = append(result, diff)
			seen[seenKey] = true
		}
	}

	// Sort result slice by environment variable name
	sort.Slice(result, func(i, j int) bool {
		return result[i][0] < result[j][0]
	})

	return result
}

func LoadEnv(envFilePath string) (EnvxMap, error) {
	content, err := os.ReadFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	return parseEnvLines(lines), nil
}

// locked 인 항목은 오버라이드를 하지 않는다.
func Override(old EnvxMap, new EnvxMap) EnvxMap {
	result := make(EnvxMap)
	for key, value := range old {
		result[key] = value
	}
	for key, value := range new {
		if existingValue, exists := result[key]; !exists || !existingValue.Locked {
			result[key] = value
		}
	}
	return result
}

func ApplyEnv(envxItemMap EnvxMap) error {
	for key, value := range envxItemMap {
		if len(key) == 0 {
			continue
		}
		err := os.Setenv(key, value.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	app := kingpin.New("dotenvx", "A command-line tool for loading environment variables from .env files")

	envFile := app.Flag("file", "Path to the .env file").Default(".env").Short('f').String()
	override := app.Flag("override", "Override existing environment variables").Short('o').Bool()
	dryRun := app.Flag("dry-run", "Print the environment variables that will be loaded").Short('d').Bool()
	verbose := app.Flag("verbose", "Print verbose output").Short('v').Bool()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	err := godotenv.Load(*envFile)
	if err == nil {
		if *override {
			parts, err := parsingPlan(*envFile)
			for _, part := range parts {
				beforeEnvs := copyEnvLines()
				beforeEnvsMap := parseEnvLines(beforeEnvs)
				err = godotenv.Overload(part)
				if err != nil {
					fmt.Printf("Error overriding environment variables: %v\n", err)
					os.Exit(1)
				}
				afterEnvs := copyEnvLines()
				afterEnvsMap := parseEnvLines(afterEnvs)
				diff := diffEnvxMaps(beforeEnvsMap, afterEnvsMap)
				if *verbose {
					fmt.Printf("Overloaded %s\n", part)
					for _, env := range diff {
						fmt.Printf("  %s\n", env)
					}
				}
			}
		}

		fmt.Printf("Successfully loaded environment variables from %s\n", *envFile)
	} else {
		fmt.Printf("Error loading %s file: %v\n", *envFile, err)
		//os.Exit(1)
	}
	if *dryRun {
		for _, env := range os.Environ() {
			fmt.Println(env)
		}
	}
}
