# video-processor

本项目(video-processor) 依赖于 [ffmpeg](https://ffmpeg.org/download.html)，需把 `ffmepg` 加入环境变量。

video-processor 可批量压缩视频文件并转为 `mp4` 格式。加上 `-watermark` 参数可批量加水印，默认在右上角，可通过 `-x` 和 `-y` 更改水印位置，或则通过 `-width` 和 `-height` 设置水印大小。

## 使用说明

把当前目录的视频文件处理为 mp4 格式并压缩：

```bash
video-processor
```

使用 `-input` 自定义输入路径，例如把当前目录的所有视频文件包括嵌套的视频文件处理完成后输出到 `../ouput`：

```bash
video-processor -input ./**/* -output ../output
```

`-watermark` 添加水印

```bash
video-processor -watermark ./watermark.png
```

## 参数

| 参数           | 类型   | 说明                                                                              |
| :------------- | :----- | :-------------------------------------------------------------------------------- |
| -concurrency   | int    | 最大并发数 (default 4)                                                            |
| -crf-threshold | int    | 文件大小阈值（字节），大于此值时添加 -crf 28 (default 10485760)                   |
| -height        | int    | 水印高度（像素），-1 表示按比例调整 (default -1)                                  |
| -input         | string | 输入路径（支持通配符，如 demo.avi, C:/video/_, C:/video/\*\*/_） (default "./\*") |
| -output        | string | 输出目录（保存处理后的视频文件） (default "./output")                             |
| -watermark     | string | 水印图片路径（可选）                                                              |
| -width         | int    | 水印宽度（像素） (default 100)                                                    |
| -x             | string | 水印水平位置（像素或表达式，如 10 或 W-w-10） (default "W-w-10")                  |
| -y             | string | 水印垂直位置（像素或表达式，如 10 或 H-h-10） (default "10")                      |

## 构建

```bash
go build -o video-processor.exe main.go
```
