#! /bin/bash


ST=$1


lwd=`pwd`
wd=`realpath ${lwd}`
td=`realpath ${lwd}/trigger/`
rd=`realpath ${lwd}/triggered`

PROJECT_ID=gjk-fat-int-3r
QUEUE_NAME=gojek-receipt-trigger
QUEUE_LOC=asia-southeast2
QUEUE_ID=projects/$PROJECT_ID/locations/$QUEUE_LOC/queues/$QUEUE_NAME
GCF_URL=https://asia-southeast2-gjk-fat-int-3r.cloudfunctions.net/tax_receipt
SERVICE_ACCOUNT_EMAIL=gjk-fat-int-3r@appspot.gserviceaccount.com

if [[ -z $TRIGGER_PROTOCOL ]]; then
  TRIGGER_PROTOCOL=http
fi 

if [[ -z $ST ]]; then
   available=`ls $td`
  ST=`echo $available | head -n1`
fi

if [[ -z $COMPANY_ID ]]; then
  COMPANY_ID=ID01 
fi 

GCS_TRIGGER=tax_receipt_trigger
PREFIX=receipts/$COMPANY_ID/$ST
temp_local_trigger_dir=$td/$ST

TIME_INCREMENT=22
MAX_CONCURRENT=10

if [[ $TRIGGER_PROTOCOL == 'http' ]]; then
  MAX_CONCURRENT=1000
fi

###============================ OPERATION ==============================

process_count=10
thread=10
GSUTILOPTIONS1="-o 'GSUtil:parallel_process_count:5' -o 'GSUtil:parallel_thread_count=2' "
GSUTILOPTIONS2="-o 'GSUtil:parallel_process_count:5' -o 'GSUtil:parallel_thread_count=3' "
GSUTILOPTIONS3="-o 'GSUtil:parallel_process_count:8' -o 'GSUtil:parallel_thread_count=4' "
GSUTILOPTIONS="-o 'GSUtil:parallel_process_count:${process_count}' -o 'GSUtil:parallel_thread_count=${thread}' "

FT=trigger_file_list.txt
if [[ ! -z $2 ]]; then
 FT=$2
fi

if [[ ! -f $FT ]]; then
  FLIST=`find $temp_local_trigger_dir -type f | sort > $FT `
fi

N=1
fcount=`wc -l $FT | cut -d' ' -f1`
tic=`date +"%Y%m%d %H%M%S"`
checktic=`date +"%Y%m%d%H%M%S" -d"+$TIME_INCREMENT minutes"`
cnt=0
while [[ -s $FT  ]] ; do
    curtic=`date +"%Y%m%d%H%M%S"`
    if [[ $curtic -gt $checktic ]]; then
      N=$(( N+1 ))
      if [[ $N -gt 4 ]]; then
        TIME_INCREMENT=60 
      fi 
      checktic=`date +"%Y%m%d%H%M%S" -d"+$TIME_INCREMENT minutes"`
    fi

    if [[ $N -gt $MAX_CONCURRENT ]];then N=$MAX_CONCURRENT; fi
    batch=$( head -$N $FT )
    lastfile=$( echo $batch | tail -1 )
    lastfile=${lastfile: -13}

    if [[ $TRIGGER_PROTOCOL == 'gcs' ]]; then
      #trigger using gcs
      head -$N $FT  | gsutil -q  $GSUTILOPTIONS -m  mv -I gs://$GCS_TRIGGER/$PREFIX/${curtic:0: -2}/ &&  sed -i -e "1,$N"d $FT 
    else
      for f in `head -$N $FT`; do
        bn=`basename $f`
        gcloud --quiet --verbosity=none --user-output-enabled false tasks create-http-task "${curtic}${bn:0: -4}" --queue $QUEUE_NAME --location $QUEUE_LOC \
          --oidc-service-account-email $SERVICE_ACCOUNT_EMAIL --oidc-token-audience $GCF_URL \
          --method PUT --url $GCF_URL --header="Content-Type: text/csv" --body-file $f \
	      &&  sed -i -e "1,$N"d $FT
      done
    fi

    cnt=$(( cnt + N ))

    echo -e  "`date`:Sending $N/$cnt/$fcount files: Last : $lastfile"
done

if  [[ ! -s $FT ]]; then

  rm -r $td/$ST/
  rm $wd/source/$ST*.csv
  #rmdir $temp_local_trigger_dir
  rm $FT 

fi
