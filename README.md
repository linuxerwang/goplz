# goplz

## Overview

goplz is a tool to help Go developers use [Please](https://please.build/) build
tool with a more flexible source file layout.

Organizing multi-project source file layout is not a trivial thing. Build tools
like Bazel, Buck, and Please provide excellent supports to layout your source
files the way it makes sense to your organization. However, for Go developers,
since Go has its own source file layout and most Go dev tools are designed on
top of this layout, it's not trivial to make both the build tools and the dev
tools to work together.

For example, some companies mandates a single source layout with a single src:

```
$TOP/ (GOPATH)
  |- src/
      |- mycompany.com/
           |- accounts/
                |- account.go
           |- orders/
           |- payments/
```

It's super simple and works for both build tools and Go dev tools. However, it
will be awkward if you company also has non-Go projects. Cluttering projects
with different language makes the layout a mess, and sometimes you might be
forced to drop this layout. Another problem is this layout is only suitable when
the whole source tree fits in one repository. For companies with huge number of
projects it's not applicable.

Therefore, the following source file layout might be more attractive:

```
$TOP/ (Go import path: mycompany.com)
  |- accounts/
       |- account.go
  |- orders/
  |- payments/
```

The benefits are:

- Projects are closer to top directory thus easier to find and more pleasant to
	work with.
- Code for a specific project can be concentrated in one directory tree.
- It doesn't mandate a Go source code layout, thus projects with other languages
	can fit in.
- The Please build tool supports this layout (in .plzconfig the "Go" section
	a ImportPath can be set).
- Cross referencing among projects are more natual. For example in
	payments/BUILD.plz:

	```python
	go_library(
		name = "payments",
		deps = [
			"//accounts/user:user",
		],
	)
	```

However, if you take this kind of source file layout, it will be hard to make
Go dev tools work, since these tools are based on the Go source file layout.
Most of the time, you need to use symlinks and environments to make the Go
toolchain happy.

This is exactly why goplz is created. It utilizes the linux FUSE library and
maps the Please source code layout to Go's standard layout, so that Go
developers can take advantage of both Go tools and Please.

It's a re-implementation of [gobazel](https://github.com/linuxerwang/gobazel)
for Please and has the same limit that it only works for Linux and MacOS users).

## Get goplz

Suppose you have a normal Go SDK, run the following command to install goplz:

```bash
$ go get https://github.com/linuxerwang/goplz
```

Ubuntu users can also download the deb package. The executable file "goplz"
must be in your $PATH or anywhere you know how to access.

## Using goplz

### Setup .plzconfig

First, you must already have a source file tree with Please as build tool.
The goplz repository itself is a good example. Note that you should have a
.plzconfig file with ImportPath set up:

```ini
; Please config file
; Leaving this file as is is enough to use plz to build your project.
; Please will stay on whatever version you currently have until you run
; 'plz update', when it will download the latest available version.
;
; Or you can uncomment the following to pin everyone to a particular version;
; when you change it all users will automatically get updated.
; [please]
; version = 14.1.12

[go]
ImportPath = github.com/linuxerwang/goplz
```

For example, if you folder is at ~/tmp/goplz, run this command:

```bash
$ plz init
```

You need to add ImportPath manually.

### Setup goplz

Next, in ~/tmp/goplz run the "goplz init" command:

```bash
$ goplz init
Initialized goplz, the virtual GOPATH is at /home/ubuntu/tmp/.goplz-gopath.
Now you can run `goplz start`.
```

The init command creates a .goplzrc file in the top folder, which sets up
VS Code as default editor. You can change it to whatever IDE you'd use.

Now you can run the start command:

```bash
$ goplz start
```

goplz starts a daemon program which creates a hidden folder .goplz-gopath, uses
it as GOPATH, where your Go dev tools should point at, and uses FUSE library to
map your real source files into this virtual GOPATH on the fly.

Also, goplz starts the configured IDE for you, with the correct virtual GOPATH
set correctly. Now ~/tmp/.goplz-gopath has the following standart Go structure:

```
.
├── bin
├── pkg
└── src
    └── github.com
        └── linuxerwang
            └── goplz
                └── pleasew
```

From now on, you change your code only in ~/tmp/.goplz-gopath, but build
your code in ~/tmp/goplz. The Go language tools should work without problem
in your IDE (autocomplete, go to definition, etc).

To stop goplz daemon, run:

```bash
$ goplz stop
```
