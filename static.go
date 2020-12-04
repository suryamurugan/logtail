// Copyright 2020 wongoo. All rights reserved.

// Code generated by "makeStatic"; DO NOT EDIT.

package logtail

var indexHTMLContent = []byte(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        * {
            margin: 0
        }

        .textBreak pre {
            white-space: pre-wrap;
            white-space: -moz-pre-wrap;
            word-wrap: break-word;
        }
    </style>
    <script>
        let maxLines = 10000;
        let lineCount = 0;
        let ws;
        let output;

        let scrollingFlag = false;

        function scrollingControl(btn) {
            scrollingFlag = !scrollingFlag;
            btn.innerText = (scrollingFlag ? "stop" : "start") + " scrolling";
        }

        let autoReconnectFlag = false;

        function autoReconnectControl(btn) {
            autoReconnectFlag = !autoReconnectFlag;
            btn.innerText = (autoReconnectFlag ? "disable" : "enable") + " auto reconnect";
            if (autoReconnectFlag && ws == null) {
                restartTail();
            }
        }

        let breakWordFlag = false;

        function breakWordControl(btn) {
            breakWordFlag = !breakWordFlag;
            if (breakWordFlag) {
                output.setAttribute("class", "textBreak");
                btn.innerText = "inline";
            } else {
                output.setAttribute("class", "");
                btn.innerText = "break word";
            }
        }

        function backgroundControl(btn) {
            let values = btn.value.split(":")
            document.body.style['background'] = values[0];
            document.body.style['color'] = values[1];
        }

        function error(message) {
            let d = document.getElementById("error_msg");
            d.innerText = message;
        }

        function print(message) {
            let d = document.createElement("pre");
            d.innerText = message;
            output.appendChild(d);

            lineCount++
            while (lineCount > maxLines) {
                output.children.item(0).remove()
                lineCount--;
            }
            if (scrollingFlag) {
                window.scrollTo(0, document.body.scrollHeight);
            }
        }

        let restartTimeout;

        function restartTail() {
            ws = null;
            error("connection closed!");

            if (autoReconnectFlag) {
                error("connection closed, reconnect in 5s!");
                if (restartTimeout) {
                    window.clearTimeout(restartTimeout);
                }
                restartTimeout = window.setTimeout(startTail, 5000);
            }
        }

        function startTail() {
            if (ws) {
                return false;
            }
            error("")
            ws = new WebSocket("ws://" + document.location.host + "/tail");
            ws.onopen = function (evt) {
            }
            ws.onclose = function (evt) {
                restartTail();
            }
            ws.onmessage = function (evt) {
                print(evt.data);
            }
            ws.onerror = function (evt) {
                error(evt.data);
                try {
                    ws.close()
                } catch (e) {
                }
                restartTail();
            }
        }

        function heartbeat() {
            if (ws != null) {
                try {
                    ws.send("ok")
                } catch (e) {
                    console.log("send error: " + e)
                }
            }
            window.setTimeout(heartbeat, 8000)
        }

        window.onload = function (evt) {
            output = document.getElementById("output");
            document.getElementById("fontSizeSel").onchange(null);
            document.getElementById("backgroundSel").onchange(null);
            document.getElementById("backgroundSel").onchange(null);
            document.getElementById("breakWordBtn").click();
            document.getElementById("scrollingBtn").click();
            document.getElementById("autoReconnectBtn").click();

            startTail();
            window.setTimeout(heartbeat, 1000);
        };
        window.onbeforeunload = function (evt) {
            if (ws != null) {
                ws.close()
            }
        }
    </script>
</head>
<body>
<div style="width: 99%; position: fixed; text-align: right;">
    <label id="error_msg" style="color: red;"></label>
    max lines:<input type="text" style="width:50px;" value="10000" onchange="maxLines=parseInt(this.value)">
    <select id="fontSizeSel" onchange="document.body.style['font-size']=this.value">
        <option value="10px">small</option>
        <option value="12px">middle</option>
        <option value="14px">big</option>
    </select>
    <select id="backgroundSel" onchange="backgroundControl(this)">
        <option value="#FFF:#000">white</option>
        <option value="#0D2A35:#64777A">black</option>
    </select>
    <button id="breakWordBtn" onclick="breakWordControl(this);"></button>
    <button id="scrollingBtn" onclick="scrollingControl(this);"></button>
    <button id="autoReconnectBtn" onclick="autoReconnectControl(this);"></button>
</div>
<div id="output"></div>
</body>
</html>`)
