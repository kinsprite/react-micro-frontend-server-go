package main

import (
	"testing"
)

func Test_matchManifestFileName(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "rmf-manifest.json",
			args: args{fileName: "rmf-manifest.json"},
			want: true,
		},
		{
			name: "rmf-manifest.v1.1.json",
			args: args{fileName: "rmf-manifest.v1.1.json"},
			want: true,
		},
		{
			name: "rmf-manifest-v1.1.json",
			args: args{fileName: "rmf-manifest-v1.1.json"},
			want: true,
		},
		{
			name: "rmf-manifest_v1.1.json",
			args: args{fileName: "rmf-manifest_v1.1.json"},
			want: true,
		},
		{
			name: "rmf-manifest-.json",
			args: args{fileName: "rmf-manifest-.json"},
			want: false,
		},
		{
			name: "2rmf-manifest-v1.1.json",
			args: args{fileName: "2rmf-manifest-v1.1.json"},
			want: false,
		},
		{
			name: "rmf-manifest-v1.1.json2",
			args: args{fileName: "rmf-manifest-v1.1.json2"},
			want: false,
		},
		{
			name: "2rmf-manifest-v1.1.json2",
			args: args{fileName: "2rmf-manifest-v1.1.json2"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchManifestFileName(tt.args.fileName); got != tt.want {
				t.Errorf("matchManifestFileName() = %v, want %v", got, tt.want)
			}
		})
	}
}
