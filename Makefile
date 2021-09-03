clean:
	rm -f bind.go
	rm -f bind_darwin_amd64.go
	rm -f bind_linux_amd64.go
	rm -f bind_windows_amd64.go
	rm -rf auto
	#astilectron-bundler cc

build:
	astilectron-bundler

start:
	"./output/linux-amd64/Warcraft_III_SLK_Edit"

debug:
	"./output/linux-amd64/Warcraft_III_SLK_Edit" -d
