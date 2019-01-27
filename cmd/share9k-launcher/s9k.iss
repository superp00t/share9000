[Setup]
AppName=SHARE9K
AppVersion=0.1
DefaultDirName={pf}\SHARE9K
DefaultGroupName=SHARE9K
UninstallDisplayIcon={app}\bin\share9k-launcher.exe
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=admin

[Files]
Source: "bin\share9k-launcher.exe"; DestDir: "{app}\bin"
Source: "version.txt"; DestDir: "{app}"

[Dirs]
Name: "{app}"; Permissions: everyone-full

[Icons]
Name: "{group}\SHARE9K"; Filename: "{app}\bin\share9k-launcher.exe"
