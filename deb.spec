deb-spec {
	control {
		pkg-name: "{{.PkgName}}"
		maintainer: "Linuxer Wang<linuxerwang@gmail.com>"
		description: "A fuse mount tool for Please to support Golang."

		other-attrs: {
			"Section": "utils",
			"Priority": "optional",
		}
	}

	{{range .Files}}
	content {
		path: "{{.Path}}"
		deb-path: "{{.DebPath}}"
	}
    {{end}}
}
