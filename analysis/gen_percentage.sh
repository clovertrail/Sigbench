if [ $# -ne 1 ]
then
	echo "Specify the input file"
	exit 1
fi
input=$1
echo "[" > /tmp/t
line=`wc -l $input |awk '{printf("%d", $1-2)}'`
head -n $line $input >> /tmp/t
endline=`wc -l $input |awk '{printf("%d", $1-1)}'`
head -n $endline $input |tail -n 1 |awk -F }, '{print $1"}"}' >> /tmp/t
echo "]" >> /tmp/t
# send,recv,sendSize,recvSize,lt_200,lt_400,lt_600,lt_800,lt_1000,lt_1200,ge_1200
go run parseresult.go -input="/tmp/t" -perc|awk '{printf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n", $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)}'
