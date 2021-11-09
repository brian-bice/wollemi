package golang

import (
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/tcncloud/wollemi/ports/golang"
)

type Package = golang.Package

func init() {
	os.Setenv("GO111MODULE", "OFF")
}

func NewImporter() *Importer {
	osarch := strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "_")
	goroot := build.Default.GOROOT

	return &Importer{
		gorootpkg: filepath.Join(goroot, "pkg", osarch),
		gorootsrc: filepath.Join(goroot, "src"),
	}
}

type Importer struct {
	gorootpkg string
	gorootsrc string
}

func (this *Importer) GOPATH() string {
	return build.Default.GOPATH
}

func (this *Importer) GOROOT() string {
	return build.Default.GOROOT
}

func (this *Importer) ModulePath(buf []byte) string {
	return modfile.ModulePath(buf)
}

func (this *Importer) ImportDir(dir string, names []string) (*Package, error) {
	out := &Package{
		GoFileImports: make(map[string][]string, len(names)),
	}

	fset := token.NewFileSet()

	for _, name := range names {
		match, err := build.Default.MatchFile(dir, name)
		if err != nil {
			return nil, err
		}

		if !match {
			out.IgnoredGoFiles = append(out.IgnoredGoFiles, name)
			continue
		}

		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}

		pkgname := file.Name.String()

		var gofiles *[]string
		var imports *[]string

		if strings.HasSuffix(name, "_test.go") {
			if strings.HasSuffix(pkgname, "_test") {
				gofiles = &out.XTestGoFiles
				imports = &out.XTestImports
				if out.Name == "" {
					out.Name = strings.TrimSuffix(pkgname, "_test")
				}
			} else {
				gofiles = &out.TestGoFiles
				imports = &out.TestImports
				if out.Name == "" {
					out.Name = pkgname
				}
			}
		} else {
			gofiles = &out.GoFiles
			imports = &out.Imports
			if out.Name == "" {
				out.Name = pkgname
			}
		}

		*gofiles = append(*gofiles, name)

		out.GoFileImports[name] = nil

	FileImports:
		for _, spec := range file.Imports {
			path := spec.Path.Value
			path = path[1 : len(path)-1]

			out.GoFileImports[name] = append(out.GoFileImports[name], path)

			for _, have := range *imports {
				if path == have {
					continue FileImports
				}
			}

			*imports = append(*imports, path)
		}
	}

	sort.Strings(out.Imports)
	sort.Strings(out.TestImports)
	sort.Strings(out.XTestImports)
	sort.Strings(out.GoFiles)
	sort.Strings(out.TestGoFiles)
	sort.Strings(out.XTestGoFiles)

	out.Goroot = strings.HasPrefix(dir, this.gorootsrc+"/")

	return out, nil
}

func (this *Importer) IsGoroot(path string) bool {
	pkg := filepath.Join(this.gorootpkg, path+".a")
	src := filepath.Join(this.gorootsrc, path)

	for _, path := range []string{pkg, src} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

func NewPackage(in *build.Package, err error) (*Package, error) {
	if err != nil {
		return nil, err
	}

	var out *Package

	if in != nil {
		out = &Package{
			Name:           in.Name,
			Goroot:         in.Goroot,
			Imports:        in.Imports,
			TestImports:    in.TestImports,
			XTestImports:   in.XTestImports,
			GoFiles:        in.GoFiles,
			TestGoFiles:    in.TestGoFiles,
			XTestGoFiles:   in.XTestGoFiles,
			IgnoredGoFiles: in.IgnoredGoFiles,
		}
	}

	return out, nil
}
