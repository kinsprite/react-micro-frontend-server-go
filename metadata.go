package main

// MetadataExtra Extra metadata
type MetadataExtra struct {
	DefaultRoute string `json:"defaultRoute"`
}

// MetadataApp App's metadata
type MetadataApp struct {
	ID           string   `json:"id"`
	Dependencies []string `json:"dependencies"` // NOT implement yet
	Entries      []string `json:"entries"`
	Routes       []string `json:"routes"`
	Render       string   `json:"render"` // render ID
}

// Metadata the all metadata
type Metadata struct {
	Apps  []MetadataApp `json:"apps"`
	Extra MetadataExtra `json:"extra"`
}

// GitRevision Git revision has tag or short SHA
type GitRevision struct {
	Tag   string `json:"tag"`
	Short string `json:"short"`
}

// AppManifest App manifest from 'rmf-manifest.json'
type AppManifest struct {
	Entrypoints []string `json:"entrypoints"` // NOT implement yet
	// Files       map[string]string `json:"files"`
	GitRevision   GitRevision `json:"gitRevision"`
	LibraryExport string      `json:"libraryExport"`
	PublicPath    string      `json:"publicPath"`
	Routes        []string    `json:"routes"`
	ServiceName   string      `json:"serviceName"`
}
