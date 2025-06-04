package types

var (
	Success             = 2000
	ClientError         = 4000
	ServerInternalError = 5000
)

var (
	FileFormatError  = "The file format is not supported. Currently, only MP4 is supported."
	CreateFileError  = "create file error"
	SaveFileError    = "save file error"
	ConvertFileError = "convert file error"
	UploadFileError  = "upload file error"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}
