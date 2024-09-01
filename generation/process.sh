#! /bin/bash

if [[ -z $1 ]]; then
 export ST=$SERVICE_TYPE
else
  export ST=$1
fi

if [[ -z $COMPANY_ID ]]; then
  COMPANY_ID=ID01 
fi 


echo "$(date) Cleaning up previous job"
rm -rf source/*
rm -rf trigger/*

GCS_SOURCE_BUCKET=enterprise_bi_exchange
GCS_SOURCE_PREFIX=tax_invoice_testing/$COMPANY_ID
echo "`date` DOWNLOAD EXTRACTED $ST FILE FROM GCS"
time gsutil -m cp "gs://$GCS_SOURCE_BUCKET/$GCS_SOURCE_PREFIX/$ST-*" source/


if [[ ! -z $SKIP ]]; then
  echo "$(date) Skipping $SKIP source files"
  for f in source/$ST* ; do
  d=${f: -9}
  d=${d:0:5}
  if [[ $(( $((10#$d)) - $SKIP )) -lt 1 ]]; then 
    rm $f ;
  fi
  done
fi

echo "`date` $ST Data statistics"
#wc -l source/$ST*.csv
TOTAL_RECORD=`wc -l source/$ST*.csv | tail -1 | cut -d\t -f1`
echo $TOTAL_RECORD

echo "`date` SPLIT DOWNLOADED $ST FILE INTO 1000 record chuncks"
for f in source/$ST*.csv; do
  ./split.sh $f
done


echo "`date` START TRIGGERING $ST files"
time ./trigger.sh $ST


echo "`date` APPROX:$TOTAL_RECORD records of $ST PROCESSED"
