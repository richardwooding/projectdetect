package projectdetect

func init() {
	Register(&ProjectType{
		Name:          "go",
		Description:   "Go module (go.mod)",
		Indicators:    []Indicator{{HasFile: "go.mod"}},
		BuildExcludes: []string{"vendor"},
	})

	Register(&ProjectType{
		Name:          "node",
		Description:   "Node.js / npm / yarn / pnpm (package.json)",
		Indicators:    []Indicator{{HasFile: "package.json"}},
		BuildExcludes: []string{"node_modules"},
	})

	Register(&ProjectType{
		Name:          "rust",
		Description:   "Rust crate (Cargo.toml)",
		Indicators:    []Indicator{{HasFile: "Cargo.toml"}},
		BuildExcludes: []string{"target"},
	})

	Register(&ProjectType{
		Name:        "python",
		Description: "Python project (pyproject.toml / requirements.txt / Pipfile / setup.py / setup.cfg)",
		Indicators: []Indicator{
			{HasFile: "pyproject.toml"},
			{HasFile: "requirements.txt"},
			{HasFile: "Pipfile"},
			{HasFile: "setup.py"},
			{HasFile: "setup.cfg"},
		},
		BuildExcludes: []string{"__pycache__", ".venv", "venv", ".tox", ".pytest_cache", ".mypy_cache", ".ruff_cache"},
	})

	Register(&ProjectType{
		Name:          "ruby",
		Description:   "Ruby Bundler project (Gemfile)",
		Indicators:    []Indicator{{HasFile: "Gemfile"}},
		BuildExcludes: []string{".bundle"},
	})

	Register(&ProjectType{
		Name:          "java-maven",
		Description:   "Java Maven project (pom.xml)",
		Indicators:    []Indicator{{HasFile: "pom.xml"}},
		BuildExcludes: []string{"target"},
	})

	Register(&ProjectType{
		Name:        "java-gradle",
		Description: "Java/Kotlin Gradle project (build.gradle / build.gradle.kts)",
		Indicators: []Indicator{
			{HasFile: "build.gradle"},
			{HasFile: "build.gradle.kts"},
			{HasFile: "settings.gradle"},
			{HasFile: "settings.gradle.kts"},
		},
		BuildExcludes: []string{"build", ".gradle"},
	})

	Register(&ProjectType{
		Name:        "dotnet",
		Description: ".NET project (*.csproj / *.fsproj / *.vbproj / *.sln / *.slnx, MSBuild + SDK markers)",
		Indicators: []Indicator{
			{HasGlob: "*.csproj"},
			{HasGlob: "*.fsproj"},
			{HasGlob: "*.vbproj"},
			{HasGlob: "*.sln"},
			// *.sln is the legacy text solution; *.slnx is the newer XML
			// solution format (VS 17.10+, `dotnet sln migrate`). filepath.Match
			// treats them as distinct suffixes, so *.slnx needs its own glob.
			{HasGlob: "*.slnx"},
			{HasGlob: "*.slnf"}, // solution filter (JSON view of a .sln)
			// Root markers that make a solution-less / .slnx-only SDK-style
			// repo root detect as dotnet even when every *.csproj lives in a
			// subdirectory. Each is .NET-exclusive. HasFile is case-insensitive
			// (equalFold), so "nuget.config" also matches NuGet.Config / NuGet.config.
			{HasFile: "global.json"},
			{HasFile: "Directory.Build.props"},
			{HasFile: "Directory.Packages.props"},
			{HasFile: "nuget.config"},
		},
		BuildExcludes: []string{"bin", "obj"},
	})

	Register(&ProjectType{
		Name:          "terraform",
		Description:   "Terraform / OpenTofu (*.tf)",
		Indicators:    []Indicator{{HasGlob: "*.tf"}},
		BuildExcludes: []string{".terraform"},
	})

	Register(&ProjectType{
		Name:        "docker-compose",
		Description: "Docker Compose stack (docker-compose.{yml,yaml} / compose.{yml,yaml})",
		Indicators: []Indicator{
			{HasFile: "docker-compose.yml"},
			{HasFile: "docker-compose.yaml"},
			{HasFile: "compose.yml"},
			{HasFile: "compose.yaml"},
		},
		// docker-compose stacks don't have a canonical artefact dir.
	})

	// Additional language / build-tool ecosystems. Indicators match a
	// directory's FILES only (HasFile / HasGlob never see subdirectory
	// names), so each marker below is a real file at the project root —
	// e.g. Swift uses Package.swift / *.podspec rather than the
	// *.xcodeproj bundle, which is a directory.

	Register(&ProjectType{
		Name:        "swift",
		Description: "Swift package (Package.swift) / CocoaPods (*.podspec)",
		Indicators: []Indicator{
			{HasFile: "Package.swift"},
			{HasGlob: "*.podspec"},
		},
		BuildExcludes: []string{".build", ".swiftpm", "DerivedData"},
	})

	Register(&ProjectType{
		Name:          "php",
		Description:   "PHP Composer project (composer.json)",
		Indicators:    []Indicator{{HasFile: "composer.json"}},
		BuildExcludes: []string{"vendor"},
	})

	Register(&ProjectType{
		Name:          "scala-sbt",
		Description:   "Scala sbt project (build.sbt)",
		Indicators:    []Indicator{{HasFile: "build.sbt"}},
		BuildExcludes: []string{"target", ".bsp"},
	})

	Register(&ProjectType{
		Name:        "scala-mill",
		Description: "Scala Mill project (build.mill / build.sc)",
		Indicators: []Indicator{
			{HasFile: "build.mill"},
			{HasFile: "build.sc"},
		},
		BuildExcludes: []string{"out"},
	})

	Register(&ProjectType{
		Name:          "cmake",
		Description:   "C/C++ CMake project (CMakeLists.txt)",
		Indicators:    []Indicator{{HasFile: "CMakeLists.txt"}},
		BuildExcludes: []string{"build", "cmake-build-debug", "cmake-build-release", "_build"},
	})

	Register(&ProjectType{
		Name:        "autotools",
		Description: "GNU Autotools project (configure.ac / configure.in / Makefile.am)",
		Indicators: []Indicator{
			{HasFile: "configure.ac"},
			{HasFile: "configure.in"},
			{HasFile: "Makefile.am"},
		},
		BuildExcludes: []string{"autom4te.cache"},
	})

	Register(&ProjectType{
		Name:        "r",
		Description: "R package / project (DESCRIPTION / *.Rproj)",
		Indicators: []Indicator{
			{HasFile: "DESCRIPTION"},
			{HasGlob: "*.Rproj"},
		},
		// R builds in-place; no canonical artefact dir.
	})

	Register(&ProjectType{
		Name:        "zig",
		Description: "Zig project (build.zig / build.zig.zon)",
		Indicators: []Indicator{
			{HasFile: "build.zig"},
			{HasFile: "build.zig.zon"},
		},
		BuildExcludes: []string{"zig-out", "zig-cache", ".zig-cache"},
	})

	Register(&ProjectType{
		Name:        "perl",
		Description: "Perl distribution (Makefile.PL / Build.PL / cpanfile / dist.ini)",
		Indicators: []Indicator{
			{HasFile: "Makefile.PL"},
			{HasFile: "Build.PL"},
			{HasFile: "cpanfile"},
			{HasFile: "dist.ini"},
		},
		BuildExcludes: []string{"blib", "_build"},
	})

	Register(&ProjectType{
		Name:        "matlab",
		Description: "MATLAB project / toolbox (*.prj)",
		Indicators:  []Indicator{{HasGlob: "*.prj"}},
		// .prj is an XML project/toolbox descriptor; MATLAB has no
		// canonical build-output dir.
	})

	// Static-site generators. The 8 most-encountered SSGs; each maps
	// to a canonical indicator file plus its standard build-output
	// directory. The is_static_site CEL family predicate (see
	// internal/celexpr/evaluator.go's staticSiteTypes) fires for any
	// of these. Hugo uses only the modern hugo.{toml,yaml,yml}
	// filenames (preferred since v0.110) — bare config.toml is too
	// ambiguous to ship as a default; legacy sites can use a custom
	// YAML config.

	Register(&ProjectType{
		Name:        "hugo",
		Description: "Hugo static site (hugo.{toml,yaml,yml})",
		Indicators: []Indicator{
			{HasFile: "hugo.toml"},
			{HasFile: "hugo.yaml"},
			{HasFile: "hugo.yml"},
		},
		BuildExcludes: []string{"public", "resources"},
	})

	Register(&ProjectType{
		Name:        "jekyll",
		Description: "Jekyll static site (_config.{yml,yaml})",
		Indicators: []Indicator{
			{HasFile: "_config.yml"},
			{HasFile: "_config.yaml"},
		},
		BuildExcludes: []string{"_site", ".jekyll-cache", ".sass-cache"},
	})

	Register(&ProjectType{
		Name:        "eleventy",
		Description: "Eleventy static site (.eleventy.js / eleventy.config.*)",
		Indicators: []Indicator{
			{HasFile: ".eleventy.js"},
			{HasFile: "eleventy.config.js"},
			{HasFile: "eleventy.config.cjs"},
			{HasFile: "eleventy.config.mjs"},
			{HasFile: "eleventy.config.ts"},
		},
		BuildExcludes: []string{"_site"},
	})

	Register(&ProjectType{
		Name:        "astro",
		Description: "Astro static site (astro.config.{mjs,cjs,js,ts})",
		Indicators: []Indicator{
			{HasFile: "astro.config.mjs"},
			{HasFile: "astro.config.cjs"},
			{HasFile: "astro.config.js"},
			{HasFile: "astro.config.ts"},
		},
		BuildExcludes: []string{"dist", ".astro"},
	})

	Register(&ProjectType{
		Name:        "gatsby",
		Description: "Gatsby static site (gatsby-config.{js,ts,mjs})",
		Indicators: []Indicator{
			{HasFile: "gatsby-config.js"},
			{HasFile: "gatsby-config.ts"},
			{HasFile: "gatsby-config.mjs"},
		},
		BuildExcludes: []string{"public", ".cache", ".gatsby"},
	})

	Register(&ProjectType{
		Name:        "mkdocs",
		Description: "MkDocs documentation site (mkdocs.{yml,yaml})",
		Indicators: []Indicator{
			{HasFile: "mkdocs.yml"},
			{HasFile: "mkdocs.yaml"},
		},
		BuildExcludes: []string{"site"},
	})

	Register(&ProjectType{
		Name:        "docusaurus",
		Description: "Docusaurus documentation site (docusaurus.config.{js,ts,mjs})",
		Indicators: []Indicator{
			{HasFile: "docusaurus.config.js"},
			{HasFile: "docusaurus.config.ts"},
			{HasFile: "docusaurus.config.mjs"},
		},
		BuildExcludes: []string{"build", ".docusaurus"},
	})

	Register(&ProjectType{
		Name:          "pelican",
		Description:   "Pelican static site (pelicanconf.py)",
		Indicators:    []Indicator{{HasFile: "pelicanconf.py"}},
		BuildExcludes: []string{"output"},
	})
}
