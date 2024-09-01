

# GOLANG Receipt Printer 

High performance pdf receipt to Google GCS API.  
Used to generate millions/billios of receipt in month end closing, after order are posted.  
( amounts are consolidated, number of order are piled up).  


The main issue are the `i/o` limitations, need to control file generation speed. It can go thousands of file per seconds and make the disk stalled. using GCS, need to ramp up GCS API request limit. Experienced to generated 30.000 files/seconds.
Put to GCS, to put it as *sharing ready file system`, since the number of files can be millions.  
Moving that numbers of files are multiple time surely will take times.  


### How to use

check on `company.go` file for your company header ( company name, tax number, and address),   
if can have multiple company, and selected by company ID parameter.


technical:
work on 2 mode, gcs trigger, or http trigger ( please look on `.gitlab-ci.yaml` for deployment info)
1. gcs mode: just put csv file on the gcs, the GCF will pull the file and generate the data. Used it to utilize gcs parallelism.
2. http mode: trigger the `google cloud function`  url with post/put method, with csv file with payload.

generation:
look for `generation` folder to see the logic of _"GCS ramp up"_



### TODO
1. rewrite the writer so it can write directly disk, it's actually does, by using `dev` in `env` ENV_VAR  
2. internal `i/o` rate controller. For now, it controlled by how many goroutine created.

reference:
* https://bytegoblin.io/blog/fundamentals-of-i-o-in-go
* https://medium.com/@halilylm/cpu-vs-i-o-bound-benchmarking-in-go-fcd4f053694e  
* https://www.toptal.com/back-end/server-side-io-performance-node-php-java-go  
* https://peabody.io/post/server-env-benchmarks/