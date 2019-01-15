REM A Batch to Set Environment on Windows for First Build
set "GOPATH=c:\tools\go"
set "Path=%GOPATH%\bin;%Path%"
setx GOPATH "%GOPATH%"
setx Path "%Path%"
