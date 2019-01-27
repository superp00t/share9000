function pack() {
  strip $1
  upx $1
}

cd ../share9k
bash build-assets.sh
echo `pwd`
cd ../share9k-launcher

BUILD_DIR="/tmp/$(uuidgen)"

DEPLOY_DIRECTORY="//JOSHUAS-IMAC/mass_storage/img/share9k"
VERS=$(cat version.txt)
mkdir $BUILD_DIR
mkdir $BUILD_DIR/bin
BEGIN_DIR=$(pwd)

rsrc.exe -arch amd64 -ico icon.ico -o ../share9k/rsrc.syso
rsrc.exe -arch amd64 -ico icon.ico -o rsrc.syso

go build -ldflags -H=windowsgui github.com/superp00t/share9000/cmd/share9k-launcher
pack share9k-launcher.exe
cp share9k-launcher.exe $BUILD_DIR/bin/
# set to zero, so launcher will download on startup
echo "0.0" > $BUILD_DIR/version.txt

cp s9k.iss $BUILD_DIR

cd $BUILD_DIR

/c/Program\ Files\ \(x86\)/Inno\ Setup\ 5/ISCC.exe s9k.iss

mv Output/mysetup.exe $BEGIN_DIR/share9k-install.exe 
cd $BEGIN_DIR
rm -rf $BUILD_DIR
BUILD_DIR=/tmp/$(uuidgen)

mkdir $BUILD_DIR
mkdir $BUILD_DIR/bin

if ! go build -ldflags "-H windowsgui" github.com/superp00t/share9000/cmd/share9k; then
 exit 0
fi
pack share9k.exe
cp share9k.exe $BUILD_DIR/bin

cp /mingw64/bin/SDL2.dll $BUILD_DIR/bin
cp version.txt $BUILD_DIR/version.txt
# cp /mingw64/lib/gdk-pixbuf-2.0/2.10.0/loaders/*.dll $BUILD_DIR/bin/

cd $BUILD_DIR
rm $BEGIN_DIR/build.zip
zip -r $BEGIN_DIR/build.zip *
cd $BEGIN_DIR
rm -rf $BUILD_DIR

cd $BEGIN_DIR
ZIP_NAME="share9k-windows-amd64-$VERS.zip"

mv build.zip $ZIP_NAME
echo | minisign -Sm $ZIP_NAME

rm $DEPLOY_DIRECTORY/*.zip
rm $DEPLOY_DIRECTORY/*.minisig
mv "$ZIP_NAME"* $DEPLOY_DIRECTORY
cp version.txt $DEPLOY_DIRECTORY
mv share9k-install.exe $DEPLOY_DIRECTORY