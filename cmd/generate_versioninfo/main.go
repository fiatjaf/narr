package main

import (
	"flag"
	"os"
	"strings"
)

var rsrc = `1 VERSIONINFO
FILEVERSION     {VERSION_COMMA},0
PRODUCTVERSION  {VERSION_COMMA},0
BEGIN
  BLOCK "StringFileInfo"
  BEGIN
    BLOCK "080904E4"
    BEGIN
      VALUE "CompanyName", "Tabajara Inc"
      VALUE "FileDescription", "Not another RSS reader"
      VALUE "FileVersion", "{VERSION}"
      VALUE "InternalName", "narr"
      VALUE "LegalCopyright", "fiatjaf"
      VALUE "OriginalFilename", "narr.exe"
      VALUE "ProductName", "narr"
      VALUE "ProductVersion", "{VERSION}"
    END
  END
  BLOCK "VarFileInfo"
  BEGIN
    VALUE "Translation", 0x809, 1252
  END
END

1 ICON "icon.ico"
`

func main() {
	var version, outfile string
	flag.StringVar(&version, "version", "0.0.0", "")
	flag.StringVar(&outfile, "outfile", "versioninfo.rc", "")
	flag.Parse()

	version_comma := strings.ReplaceAll(version, ".", ",")

	out := strings.ReplaceAll(rsrc, "{VERSION}", version)
	out = strings.ReplaceAll(out, "{VERSION_COMMA}", version_comma)

	os.WriteFile(outfile, []byte(out), 0644)
}
