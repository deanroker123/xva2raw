# xva-raw

Utility to convert XVA files to Raw disk images. Similar to [xva-img](https://github.com/eriklax/xva-img/) but without the need to extract the XVA file first.

**Currently only supports first disk image found in the file**

Only uses the standard go library so just run

`go get github.com/deanroker123/xva2raw`

change to the directory and run

`go install`

get all the goodness

run it like so

`./xva2raw <name of xva file> <name of raw images>`

then sit back and wait. Runs best with source and destination images on 2 drives. Takes about 1:30h for a 600GB disk image on with source on external USB3 SSD and destination on USB3 spinning disk.

Reading and Writing are handled by separate go routines. If there are lots of empty blocks in the XVA file it can sit at __Waiting for writing to finish__ for some time, especially if the destination disk is slower than the source.

