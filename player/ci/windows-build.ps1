$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true
New-Item -ItemType Directory -Force dist | Out-Null

go vet ./...
go test ./...

$env:CGO_ENABLED = '0'
go build -ldflags '-H=windowsgui -s -w' -o dist/bonzi.exe .
