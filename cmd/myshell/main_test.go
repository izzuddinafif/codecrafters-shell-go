/*
This is Afif's Implementation of Shell.
*/
package main

import "testing"

func Test_findLongestCommonPrefix(t *testing.T) {
	tests := []struct {
		name    string
		match   string
		matches []string
		want    string
	}{
		// {
		// 	name:    "Empty slice",
		// 	matches: []string{},
		// 	want:    "",
		// },
		{
			name:    "Single string",
			match:   "hello",
			matches: []string{"hello"},
			want:    "hello",
		},
		{
			name:    "Common prefix exists",
			match:   "abc_",
			matches: []string{"abc_def", "abc_def_ghi", "abc_def_ghi_jkl"},
			want:    "abc_def",
		},
		{
			name:    "No common prefix",
			match:   "dog",
			matches: []string{"dog", "racecar", "car"},
			want:    "dog",
		},
		{
			name:    "All strings identical",
			match:   "ab",
			matches: []string{"abc", "abc_def", "abc_def_ghi_jkl"},
			want:    "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findLongestCommonPrefix(tt.match, tt.matches); got != tt.want {
				t.Errorf("findLongestCommonPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
