package godotenvx

import (
	"reflect"
	"testing"
)

func TestParseEnvLines(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected EnvMap
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: EnvMap{},
		},
		{
			name:  "single valid key-value pair",
			input: []string{"KEY1=value1"},
			expected: EnvMap{
				"KEY1": {Key: "KEY1", Value: "value1", Locked: false},
			},
		},
		{
			name:  "multiple valid key-value pairs",
			input: []string{"KEY1=value1", "KEY2=value2"},
			expected: EnvMap{
				"KEY1": {Key: "KEY1", Value: "value1", Locked: false},
				"KEY2": {Key: "KEY2", Value: "value2", Locked: false},
			},
		},
		{
			name:  "key with locked",
			input: []string{"# LOCK", "KEY1=value1"},
			expected: EnvMap{
				"KEY1": {Key: "KEY1", Value: "value1", Locked: true},
			},
		},
		{
			name:  "key with inline comment",
			input: []string{"KEY1=value1 # some comment"},
			expected: EnvMap{
				"KEY1": {Key: "KEY1", Value: "value1 # some comment", Locked: false},
			},
		},
		{
			name:     "comment-only lines",
			input:    []string{"# this is a comment", "  ", "# another comment"},
			expected: EnvMap{},
		},
		{
			name: "mixed valid lines and comments",
			input: []string{
				"# Header comment",
				"KEY1=value1",
				"# LOCK",
				"KEY2=value2",
				"# Comment after lock",
				"KEY3=value3",
			},
			expected: EnvMap{
				"KEY1": {Key: "KEY1", Value: "value1", Locked: false},
				"KEY2": {Key: "KEY2", Value: "value2", Locked: true},
				"KEY3": {Key: "KEY3", Value: "value3", Locked: false},
			},
		},
		{
			name:     "invalid key-value format",
			input:    []string{"INVALID_LINE"},
			expected: EnvMap{},
		},
		{
			name:  "valid and invalid lines mixed",
			input: []string{"KEY1=value1", "INVALID_LINE", "KEY2=value2"},
			expected: EnvMap{
				"KEY1": {Key: "KEY1", Value: "value1", Locked: false},
				"KEY2": {Key: "KEY2", Value: "value2", Locked: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewEnvMapFromEnviron(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseEnvLines(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// override 함수에 대한 단위 테스트 코드
// 기존 환경변수와 새 환경변수를 병합할 때
// locked가 true인 항목이 기존에 있으면 오버라이드 하지 않고 보존하는지 검증
func TestOverride(t *testing.T) {
	tests := []struct {
		name     string
		old      EnvMap
		new      EnvMap
		expected EnvMap
	}{
		{
			name:     "empty maps",
			old:      EnvMap{},
			new:      EnvMap{},
			expected: EnvMap{},
		},
		{
			name: "add new keys",
			old: EnvMap{
				"KEY1": {Value: "value1", Locked: false},
			},
			new: EnvMap{
				"KEY2": {Value: "value2", Locked: false},
			},
			expected: EnvMap{
				"KEY1": {Value: "value1", Locked: false},
				"KEY2": {Value: "value2", Locked: false},
			},
		},
		{
			name: "override unlocked key",
			old: EnvMap{
				"KEY1": {Value: "value1", Locked: false},
			},
			new: EnvMap{
				"KEY1": {Value: "new_value", Locked: false},
			},
			expected: EnvMap{
				"KEY1": {Value: "new_value", Locked: false},
			},
		},
		{
			name: "preserve locked key",
			old: EnvMap{
				"KEY1": {Value: "value1", Locked: true},
			},
			new: EnvMap{
				"KEY1": {Value: "new_value", Locked: false},
			},
			expected: EnvMap{
				"KEY1": {Value: "value1", Locked: true},
			},
		},
		{
			name: "mixed keys",
			old: EnvMap{
				"KEY1": {Value: "value1", Locked: true},
				"KEY2": {Value: "value2", Locked: false},
			},
			new: EnvMap{
				"KEY2": {Value: "new_value2", Locked: false},
				"KEY3": {Value: "value3", Locked: false},
			},
			expected: EnvMap{
				"KEY1": {Value: "value1", Locked: true},
				"KEY2": {Value: "new_value2", Locked: false},
				"KEY3": {Value: "value3", Locked: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := override(tt.old, tt.new)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Override() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDiffEnvs(t *testing.T) {
	tests := []struct {
		name     string
		before   []string
		after    []string
		expected [][3]string
	}{
		{
			name:     "no changes",
			before:   []string{"KEY1=value1", "KEY2=value2"},
			after:    []string{"KEY1=value1", "KEY2=value2"},
			expected: [][3]string{},
		},
		{
			name:   "new key added",
			before: []string{"KEY1=value1"},
			after:  []string{"KEY1=value1", "KEY2=value2"},
			expected: [][3]string{
				{"KEY2", "", "value2"},
			},
		},
		{
			name:   "key removed",
			before: []string{"KEY1=value1", "KEY2=value2"},
			after:  []string{"KEY1=value1"},
			expected: [][3]string{
				{"KEY2", "value2", ""},
			},
		},
		{
			name:   "key value changed",
			before: []string{"KEY1=value1"},
			after:  []string{"KEY1=updated_value"},
			expected: [][3]string{
				{"KEY1", "value1", "updated_value"},
			},
		},
		{
			name:   "multiple changes",
			before: []string{"KEY1=value1", "KEY2=value2"},
			after:  []string{"KEY1=updated_value", "KEY3=value3"},
			expected: [][3]string{
				{"KEY1", "value1", "updated_value"},
				{"KEY2", "value2", ""},
				{"KEY3", "", "value3"},
			},
		},
		{
			name:     "empty before and after",
			before:   []string{},
			after:    []string{},
			expected: [][3]string{},
		},
		{
			name:   "empty before, non-empty after",
			before: []string{},
			after:  []string{"KEY1=value1"},
			expected: [][3]string{
				{"KEY1", "", "value1"},
			},
		},
		{
			name:   "non-empty before, empty after",
			before: []string{"KEY1=value1"},
			after:  []string{},
			expected: [][3]string{
				{"KEY1", "value1", ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeMap := NewEnvMapFromEnviron(tt.before)
			afterMap := NewEnvMapFromEnviron(tt.after)
			result := DiffEnvMap(beforeMap, afterMap)
			print("1111", result)
			if len(result) == 0 && len(tt.expected) == 0 {
				// Both slices are empty, test passes
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("diffEnvxMaps() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestParsingPlan(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "empty file path",
			input:    "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "single file without env extension",
			input:    "/path/to/file.txt",
			expected: []string{"/path/to/file.txt"},
			wantErr:  false,
		},
		{
			name:     "single env file",
			input:    "/path/to/.env",
			expected: []string{"/path/to/.env"},
			wantErr:  false,
		},
		{
			name:  "multiple parts with env extension",
			input: "/path/to/.env.local",
			expected: []string{
				"/path/to/.env",
				"/path/to/.env.local",
			},
			wantErr: false,
		},
		{
			name:  "multiple parts with env in the middle",
			input: "/path/to/config.env.development",
			expected: []string{
				"/path/to/config.env",
				"/path/to/config.env.development",
			},
			wantErr: false,
		},
		{
			name:  "longer chain with env extension",
			input: "/path/to/app.config.env.production.api",
			expected: []string{
				"/path/to/app.config.env",
				"/path/to/app.config.env.production",
				"/path/to/app.config.env.production.api",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := OverridePlan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsingPlan() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parsingPlan() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
