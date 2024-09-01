#! /bin/bash

file=`realpath $1`
b=`basename $1`
fileId=${b: -9}
fileId=${fileId:0:5}
trigger_path="trigger/$ST/$fileId"
mkdir -p $trigger_path
split -l 1000 -d -a 4 --additional-suffix=.csv $file ${trigger_path}/
header=/tmp/header$$.txt
head -n1 $file > $header
c=0

#echo $trigger_path
for sf in $trigger_path/*.csv ; do
  if [[ $sf == "*0000.csv" ]]; then
    continue
  fi
 # echo $sf
  sed -i "1i`cat $header`" $sf
  ((c++))
done

echo $file:$sf:$c

rm $header
