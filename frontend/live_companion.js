let liveWs = null;
let inputAudioCtx = null;  // 16kHz for mic input
let outputAudioCtx = null; // 24kHz for playback
let stream = null;
let processor = null;

let isAssistantActive = false;

// Convert base64 from Gemini to ArrayBuffer
function base64ToArrayBuffer(base64) {
    const binary_string = window.atob(base64);
    const len = binary_string.length;
    const bytes = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
        bytes[i] = binary_string.charCodeAt(i);
    }
    return bytes.buffer;
}

// Convert ArrayBuffer to Float32Array (Gemini sends 16-bit PCM at 24000Hz)
function pcm16ToFloat32(buffer) {
    const int16 = new Int16Array(buffer);
    const float32 = new Float32Array(int16.length);
    for (let i = 0; i < int16.length; i++) {
        float32[i] = int16[i] / 0x7FFF;
    }
    return float32;
}

// Convert Float32Array to 16-bit PCM for Gemini (Gemini expects 16-bit PCM at 16000Hz)
function float32ToPCM16Base64(float32Array) {
    const int16 = new Int16Array(float32Array.length);
    for (let i = 0; i < float32Array.length; i++) {
        const s = Math.max(-1, Math.min(1, float32Array[i]));
        int16[i] = s < 0 ? s * 0x8000 : s * 0x7FFF;
    }
    const bytes = new Uint8Array(int16.buffer);
    let binary = '';
    for (let i = 0; i < bytes.byteLength; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return window.btoa(binary);
}

// Audio playback queue
let audioQueue = [];
let nextPlayTime = 0;
let currentSource = null;

// Stop Gemini Playback on Barge-in
function stopPlayback() {
    if (currentSource) {
        try {
            currentSource.stop();
        } catch(e) {}
        currentSource.disconnect();
        currentSource = null;
    }
    nextPlayTime = 0;
}

function playAudioData(base64PCM) {
    if (!outputAudioCtx) return;
    
    try {
        const buffer = base64ToArrayBuffer(base64PCM);
        const float32Data = pcm16ToFloat32(buffer);
        
        // Gemini audio outputs at 24kHz
        const audioBuffer = outputAudioCtx.createBuffer(1, float32Data.length, 24000);
        audioBuffer.getChannelData(0).set(float32Data);
        
        const source = outputAudioCtx.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(outputAudioCtx.destination);
        
        if (nextPlayTime < outputAudioCtx.currentTime) {
            nextPlayTime = outputAudioCtx.currentTime + 0.05;
        }
        source.start(nextPlayTime);
        nextPlayTime += audioBuffer.duration;
        currentSource = source;
    } catch(err) {
        console.error("Audio playback error:", err);
    }
}

async function toggleVoiceAssistant() {
    const btn = document.getElementById("voice-anim");
    const statusText = document.getElementById("voice-text");
    
    if (isAssistantActive) {
        stopVoiceAssistant();
        return;
    }

    try {
        isAssistantActive = true;
        btn.innerHTML = "🔄";
        btn.style.background = "#fbbf24"; 
        btn.style.boxShadow = "none";
        statusText.innerHTML = "Connecting to AI...";

        // 1. Setup Audio Contexts
        inputAudioCtx = new (window.AudioContext || window.webkitAudioContext)({ sampleRate: 16000 });
        outputAudioCtx = new (window.AudioContext || window.webkitAudioContext)({ sampleRate: 24000 });
        nextPlayTime = outputAudioCtx.currentTime;
        
        // 2. Setup WebSocket
        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        liveWs = new WebSocket(`${protocol}//${window.location.host}/api/live/connect`);
        
        liveWs.onopen = () => {
             // 3. Send setup message
             const setupMsg = {
                 setup: {
                     model: "models/gemini-2.5-flash-native-audio-latest",
                     generationConfig: {
                         responseModalities: ["AUDIO"]
                     },
                     systemInstruction: {
                         parts: [{
                             text: "You are AsthmaCare Voice Assistant. You are talking to an asthma patient who may be experiencing symptoms. Keep responses extremely short (1-2 sentences max), calm, and conversational. Guide them through using their reliever inhaler. Speak softly and reassuringly."
                         }]
                     }
                 }
             };
             liveWs.send(JSON.stringify(setupMsg));
             
             // 4. Capture Microphone
             navigator.mediaDevices.getUserMedia({ audio: { channelCount: 1, sampleRate: 16000 } })
                .then(s => {
                    stream = s;
                    const source = inputAudioCtx.createMediaStreamSource(stream);
                    
                    // ScriptProcessor handles raw PCM chunking
                    processor = inputAudioCtx.createScriptProcessor(4096, 1, 1);
                    
                    processor.onaudioprocess = (e) => {
                         if (!isAssistantActive || liveWs.readyState !== WebSocket.OPEN) return;
                         
                         const channelData = e.inputBuffer.getChannelData(0);
                         const b64 = float32ToPCM16Base64(channelData);
                         
                         const clientContent = {
                             realtimeInput: {
                                 mediaChunks: [{
                                     mimeType: "audio/pcm;rate=16000",
                                     data: b64
                                 }]
                             }
                         };
                         liveWs.send(JSON.stringify(clientContent));
                    };
                    
                    source.connect(processor);
                    processor.connect(inputAudioCtx.destination);
                    
                    btn.innerHTML = "🎙️";
                    btn.style.background = "#10b981"; // green
                    btn.style.boxShadow = "0 4px 15px rgba(16, 185, 129, 0.4)";
                    btn.style.animation = "pulse 1.5s infinite";
                    statusText.innerHTML = "AI is listening... Speak your symptoms.";
                    
                    // Optional: trigger the first message from AI
                    const greeting = {
                        clientContent: {
                            turns: [{
                                role: "user",
                                parts: [{text: "Hello, I am having trouble breathing. Start by greeting me and asking how I feel."}]
                            }],
                            turnComplete: true
                        }
                    };
                    liveWs.send(JSON.stringify(greeting));
                })
                .catch(err => {
                    console.error("Mic error", err);
                    stopVoiceAssistant();
                    if(window.showToast) window.showToast("Microphone access denied", "error");
                });
        };
        
        liveWs.onmessage = async (e) => {
            try {
                let rawData;
                if (e.data instanceof Blob) {
                    rawData = await e.data.text();
                } else {
                    rawData = e.data;
                }
                
                const msg = JSON.parse(rawData);
                
                if (msg.setupComplete) return;
                
                if (msg.serverContent && msg.serverContent.modelTurn) {
                    const parts = msg.serverContent.modelTurn.parts;
                    if (parts && parts.length > 0) {
                        parts.forEach(p => {
                            if (p.inlineData && p.inlineData.data) {
                                playAudioData(p.inlineData.data);
                            }
                        });
                    }
                }
                
                if (msg.serverContent && msg.serverContent.interrupted) {
                    stopPlayback();
                }
                
            } catch(err) {
                console.error("Error parsing WS msg from Gemini", err);
            }
        };
        
        liveWs.onerror = (err) => {
            console.error("WebSocket error", err);
        };
        
        liveWs.onclose = () => {
            console.log("WebSocket connection closed");
            stopVoiceAssistant();
        };

    } catch (e) {
        console.error("Error setting up Gemini Live", e);
        stopVoiceAssistant();
    }
}

function stopVoiceAssistant() {
    isAssistantActive = false;
    const btn = document.getElementById("voice-anim");
    const statusText = document.getElementById("voice-text");
    
    if (btn) {
        btn.innerHTML = "🎤";
        btn.style.background = "#e11d48";
        btn.style.boxShadow = "0 4px 15px rgba(225, 29, 72, 0.4)";
        btn.style.animation = "none";
        statusText.innerHTML = "Click to talk to your emergency AI guide";
    }
    
    stopPlayback();
    
    if (processor) {
        processor.disconnect();
        processor = null;
    }
    if (stream) {
        stream.getTracks().forEach(t => t.stop());
        stream = null;
    }
    if (inputAudioCtx) {
        inputAudioCtx.close();
        inputAudioCtx = null;
    }
    if (outputAudioCtx) {
        outputAudioCtx.close();
        outputAudioCtx = null;
    }
    if (liveWs && liveWs.readyState === WebSocket.OPEN) {
        liveWs.close();
        liveWs = null;
    }
}

// Add CSS pulse animation for active recording
const style = document.createElement('style');
style.innerHTML = `
@keyframes pulse {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
  70% { transform: scale(1.05); box-shadow: 0 0 0 10px rgba(16, 185, 129, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
}`;
document.head.appendChild(style);
