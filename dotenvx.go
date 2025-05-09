package godotenvx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type EnvItem struct {
	Key    string
	Value  string
	Locked bool
}
type EnvMap map[string]*EnvItem
type Environ []string
type EnvDiffItem [3]string
type EnvDiff []EnvDiffItem

func newEnvItemFromLine(line string, locked bool) (*EnvItem, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) == 2 {
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if len(k) == 0 {
			return nil, errors.New("invalid key")
		}
		return &EnvItem{
			Key:    k,
			Value:  v,
			Locked: locked,
		}, nil
	}
	return nil, errors.New("invalid line")
}

func isFullComment(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "#")
}

func OverridePlan(envFilePath string) ([]string, error) {
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

func copyEnviron() Environ {
	environ := os.Environ()
	result := make(Environ, len(environ))
	copy(result, environ)
	return result
}

// 풀라인 주석으로 # LOCK 바로 다음 ITEM은 locked=true가 된다.
func NewEnvMapFromEnviron(environ Environ) EnvMap {
	result := make(EnvMap)

	locked := false
	for _, envLine := range environ {
		if isFullComment(envLine) {
			if strings.HasPrefix(envLine, "# LOCK") {
				locked = true
			}
			continue
		}
		envxItem, err := newEnvItemFromLine(envLine, locked)
		if err != nil {
			continue
		}
		result[envxItem.Key] = envxItem
		locked = false
	}
	return result
}

func (beforeMap EnvMap) GetDiff(afterMap EnvMap) EnvDiff {
	var result EnvDiff
	seen := make(map[string]bool)

	// left direction
	for key, value := range afterMap {
		beforeValue := ""
		if beforeMap[key] != nil {
			beforeValue = beforeMap[key].Value
		}

		if beforeValue != value.Value {
			diff := EnvDiffItem{key, beforeValue, value.Value}
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
			diff := EnvDiffItem{key, value.Value, afterValue}
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

func NewEnvMapFromFile(envFilePath string) (EnvMap, error) {
	content, err := os.ReadFile(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	return NewEnvMapFromEnviron(lines), nil
}

// locked 인 항목은 오버라이드를 하지 않는다.
func (old EnvMap) override(new EnvMap) EnvMap {
	result := make(EnvMap)
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

func (envMap EnvMap) ApplyEnviron() error {
	for key, value := range envMap {
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

func (envMap EnvMap) GetEnviron() Environ {
	result := make(Environ, 0, len(envMap))
	for key, value := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", key, value.Value))
	}
	return result
}

func LoadEnvFile(envFilePath string, override bool, verbose bool) (EnvMap, error) {
	overridePlan, err := OverridePlan(envFilePath)
	if err != nil {
		return nil, err
	}
	envMap := NewEnvMapFromEnviron(copyEnviron())
	if override {
		for _, envFilePath2 := range overridePlan {
			_, _ = fmt.Fprintf(os.Stderr, "---> Loading %s\n", envFilePath2)
			newEnvMap, err := NewEnvMapFromFile(envFilePath2)
			if err != nil {
				return nil, err
			}
			if verbose {
				diff := envMap.GetDiff(newEnvMap)
				for _, item := range diff {
					_, _ = fmt.Fprintf(os.Stderr, "     Overrided %s: %s -> %s\n", item[0], item[1], item[2])
				}
			}
			envMap = envMap.override(newEnvMap)
		}
	}
	return envMap, nil
}
