ALGO=$*
cd go-randomx
./build.sh $ALGO
cd ..

flag="-X main.BuildVersion=`echo $ALGO`"
go build -ldflags "$flag" -v  .
mkdir -p $ALGO
mv gxminer* $ALGO/