package com.vrooli.devicecontrol.capture;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.Service;
import android.content.Intent;
import android.media.AudioAttributes;
import android.media.AudioFormat;
import android.media.AudioPlaybackCaptureConfiguration;
import android.media.AudioRecord;
import android.media.MediaRecorder;
import android.media.projection.MediaProjection;
import android.media.projection.MediaProjectionManager;
import android.os.Build;
import android.os.Environment;
import android.os.Handler;
import android.os.IBinder;
import android.os.Process;
import android.os.SystemClock;

import org.json.JSONObject;

import java.io.File;
import java.io.FileOutputStream;
import java.io.RandomAccessFile;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.util.concurrent.atomic.AtomicBoolean;

/** Bounded Android playback-audio capture, with explicit projection consent per session. */
public final class CaptureService extends Service {
    static final String ACTION_START = "com.vrooli.devicecontrol.capture.START";
    static final String ACTION_STOP = "com.vrooli.devicecontrol.capture.STOP";
    static final String EXTRA_RESULT_CODE = "projection_result_code";
    static final String EXTRA_RESULT_DATA = "projection_result_data";
    private static final String CHANNEL_ID = "device-control-audio-capture";
    private static final int NOTIFICATION_ID = 7413;
    private static final int SAMPLE_RATE = 48000;
    private static final int CHANNELS = 2;
    private static final int MAX_DURATION_SECONDS = 40;

    private final AtomicBoolean stopping = new AtomicBoolean(false);
    private volatile String sessionId;
    private volatile File sessionDirectory;
    private volatile MediaProjection projection;
    private volatile AudioRecord audioRecord;
    private volatile Thread captureThread;
    private volatile long capturedBytes;
    private volatile double sumSquares;
    private volatile long sampleCount;
    private volatile int peak;
    private volatile long firstSampleElapsedNanos;

    @Override public void onCreate() {
        super.onCreate();
        createNotificationChannel();
    }

    @Override public int onStartCommand(Intent intent, int flags, int startId) {
        if (intent == null) return START_NOT_STICKY;
        String action = intent.getAction();
        String requestedSession = intent.getStringExtra(CaptureActivity.EXTRA_SESSION);
        if (ACTION_STOP.equals(action)) {
            if (requestedSession != null && requestedSession.equals(sessionId)) {
                stopCapture("stopped", "Capture stopped by Device Control.");
            } else {
                stopSelf(startId);
            }
            return START_NOT_STICKY;
        }
        if (!ACTION_START.equals(action) || requestedSession == null || !requestedSession.matches("[a-zA-Z0-9-]{1,80}") || sessionId != null) {
            stopSelf(startId);
            return START_NOT_STICKY;
        }
        sessionId = requestedSession;
        startForeground(NOTIFICATION_ID, notification("Starting playback-audio capture"));
        File musicDirectory = getExternalFilesDir(Environment.DIRECTORY_MUSIC);
        if (musicDirectory == null) {
            writeStatus(null, "failed", "Android did not provide app-specific storage.", 0L, 0L, 0, 0.0);
            stopSelf(startId);
            return START_NOT_STICKY;
        }
        sessionDirectory = new File(musicDirectory, "DeviceControlAudio/" + sessionId);
        if (!sessionDirectory.mkdirs() && !sessionDirectory.isDirectory()) {
            writeStatus(sessionDirectory, "failed", "Could not create the private capture directory.", 0L, 0L, 0, 0.0);
            stopSelf(startId);
            return START_NOT_STICKY;
        }
        try {
            int resultCode = intent.getIntExtra(EXTRA_RESULT_CODE, 0);
            Intent resultData = intent.getParcelableExtra(EXTRA_RESULT_DATA);
            MediaProjectionManager manager = getSystemService(MediaProjectionManager.class);
            if (manager == null || resultData == null) throw new IllegalStateException("Projection consent data is unavailable.");
            projection = manager.getMediaProjection(resultCode, resultData);
            projection.registerCallback(new MediaProjection.Callback() {
                @Override public void onStop() { stopCapture("projection_stopped", "Android ended the projection session."); }
            }, new Handler(getMainLooper()));
            startAudioCapture();
        } catch (Exception error) {
            writeStatus(sessionDirectory, "failed", safeMessage(error), 0L, 0L, 0, 0.0);
            stopCapture("failed", safeMessage(error));
        }
        return START_NOT_STICKY;
    }

    private void startAudioCapture() {
        AudioFormat format = new AudioFormat.Builder()
                .setEncoding(AudioFormat.ENCODING_PCM_16BIT)
                .setSampleRate(SAMPLE_RATE)
                .setChannelMask(AudioFormat.CHANNEL_IN_STEREO)
                .build();
        AudioPlaybackCaptureConfiguration config = new AudioPlaybackCaptureConfiguration.Builder(projection)
                .addMatchingUsage(AudioAttributes.USAGE_MEDIA)
                .addMatchingUsage(AudioAttributes.USAGE_GAME)
                .addMatchingUsage(AudioAttributes.USAGE_UNKNOWN)
                .build();
        int minimum = AudioRecord.getMinBufferSize(SAMPLE_RATE, AudioFormat.CHANNEL_IN_STEREO, AudioFormat.ENCODING_PCM_16BIT);
        if (minimum <= 0) throw new IllegalStateException("This device cannot configure playback audio capture.");
        int bufferBytes = Math.max(minimum * 2, SAMPLE_RATE / 5);
        audioRecord = new AudioRecord.Builder()
                .setAudioFormat(format)
                .setAudioPlaybackCaptureConfig(config)
                .setBufferSizeInBytes(bufferBytes)
                .build();
        if (audioRecord.getState() != AudioRecord.STATE_INITIALIZED) throw new IllegalStateException("Android did not initialize playback audio capture.");
        File output = new File(sessionDirectory, "playback.wav");
        long startingElapsed = SystemClock.elapsedRealtimeNanos();
        audioRecord.startRecording();
        if (audioRecord.getRecordingState() != AudioRecord.RECORDSTATE_RECORDING) throw new IllegalStateException("Android did not start playback audio capture.");
        writeStatus(sessionDirectory, "recording", "Playback capture is active; source applications may restrict capture.", startingElapsed, 0L, 0, 0.0);
        startForeground(NOTIFICATION_ID, notification("Recording playback audio"));
        captureThread = new Thread(() -> captureLoop(output, bufferBytes), "device-control-audio-capture");
        captureThread.start();
        new Handler(getMainLooper()).postDelayed(() -> {
            if (!stopping.get()) stopCapture("duration_limit", "Capture reached its 40 second safety limit.");
        }, MAX_DURATION_SECONDS * 1000L);
    }

    private void captureLoop(File output, int bufferBytes) {
        Process.setThreadPriority(Process.THREAD_PRIORITY_AUDIO);
        short[] samples = new short[Math.max(1024, bufferBytes / 2)];
        long startElapsed = 0L;
        try (FileOutputStream stream = new FileOutputStream(output, false)) {
            stream.write(new byte[44]);
            while (!stopping.get()) {
                int count = audioRecord.read(samples, 0, samples.length, AudioRecord.READ_BLOCKING);
                if (count < 0) throw new IllegalStateException("Android audio capture read failed: " + count);
                if (count == 0) continue;
                long blockElapsed = SystemClock.elapsedRealtimeNanos();
                if (startElapsed == 0L) {
                    startElapsed = blockElapsed - (long) (count / CHANNELS) * 1_000_000_000L / SAMPLE_RATE;
                    firstSampleElapsedNanos = startElapsed;
                }
                ByteBuffer pcm = ByteBuffer.allocate(count * 2).order(ByteOrder.LITTLE_ENDIAN);
                for (int i = 0; i < count; i++) {
                    int sample = samples[i];
                    peak = Math.max(peak, Math.abs(sample));
                    sumSquares += (double) sample * sample;
                    sampleCount++;
                    pcm.putShort((short) sample);
                }
                stream.write(pcm.array());
                capturedBytes += pcm.capacity();
            }
            stream.flush();
            writeWaveHeader(output, capturedBytes);
            double rms = sampleCount == 0 ? 0.0 : Math.sqrt(sumSquares / sampleCount) / 32768.0;
            double dbfs = rms <= 0.0 ? -120.0 : 20.0 * Math.log10(rms);
            writeStatus(sessionDirectory, "stopped", "Playback audio stream finalized.", startElapsed, firstSampleElapsedNanos, peak, dbfs);
        } catch (Exception error) {
            writeStatus(sessionDirectory, "failed", safeMessage(error), startElapsed, firstSampleElapsedNanos, peak, -120.0);
        }
    }

    private void stopCapture(String state, String detail) {
        if (!stopping.compareAndSet(false, true)) return;
        AudioRecord recorder = audioRecord;
        if (recorder != null) {
            try { recorder.stop(); } catch (RuntimeException ignored) { }
        }
        Thread thread = captureThread;
        if (thread != null && thread != Thread.currentThread()) {
            try { thread.join(3000L); } catch (InterruptedException interrupted) { Thread.currentThread().interrupt(); }
        }
        if (thread == null) writeStatus(sessionDirectory, state, detail, 0L, 0L, 0, 0.0);
        if (recorder != null) recorder.release();
        audioRecord = null;
        MediaProjection current = projection;
        projection = null;
        if (current != null) {
            try { current.stop(); } catch (RuntimeException ignored) { }
        }
        stopProjectionAndService();
    }

    private void stopProjectionAndService() {
        stopForeground(STOP_FOREGROUND_REMOVE);
        stopSelf();
    }

    private void createNotificationChannel() {
        NotificationChannel channel = new NotificationChannel(CHANNEL_ID, "Device Control capture", NotificationManager.IMPORTANCE_LOW);
        getSystemService(NotificationManager.class).createNotificationChannel(channel);
    }

    private Notification notification(String title) {
        return new Notification.Builder(this, CHANNEL_ID)
                .setSmallIcon(android.R.drawable.ic_menu_camera)
                .setContentTitle("Device Control")
                .setContentText(title)
                .setOngoing(true)
                .build();
    }

    static void writeStatus(File directory, String state, String detail, long captureStartElapsedNanos,
                            long firstSampleElapsedNanos, int peak, double rmsDbfs) {
        if (directory == null) return;
        try {
            if (!directory.exists() && !directory.mkdirs()) return;
            JSONObject value = new JSONObject();
            value.put("state", state);
            value.put("detail", detail);
            value.put("capture_start_elapsed_nanos", captureStartElapsedNanos);
            value.put("first_sample_elapsed_nanos", firstSampleElapsedNanos);
            value.put("peak", peak);
            value.put("rms_dbfs", rmsDbfs);
            File temporary = new File(directory, "status.json.tmp");
            File target = new File(directory, "status.json");
            try (FileOutputStream output = new FileOutputStream(temporary)) {
                output.write(value.toString().getBytes(java.nio.charset.StandardCharsets.UTF_8));
                output.getFD().sync();
            }
            if (!temporary.renameTo(target)) throw new IllegalStateException("Could not publish capture status.");
        } catch (Exception ignored) { }
    }

    private static void writeWaveHeader(File output, long dataBytes) throws Exception {
        long byteRate = SAMPLE_RATE * CHANNELS * 2L;
        ByteBuffer header = ByteBuffer.allocate(44).order(ByteOrder.LITTLE_ENDIAN);
        header.put(new byte[]{'R','I','F','F'}).putInt((int) (36 + dataBytes)).put(new byte[]{'W','A','V','E'});
        header.put(new byte[]{'f','m','t',' '}).putInt(16).putShort((short) 1).putShort((short) CHANNELS);
        header.putInt(SAMPLE_RATE).putInt((int) byteRate).putShort((short) (CHANNELS * 2)).putShort((short) 16);
        header.put(new byte[]{'d','a','t','a'}).putInt((int) dataBytes);
        try (RandomAccessFile file = new RandomAccessFile(output, "rw")) {
            file.seek(0L);
            file.write(header.array());
            file.getFD().sync();
        }
    }

    private static String safeMessage(Exception error) {
        String message = error.getMessage();
        return message == null || message.trim().isEmpty() ? error.getClass().getSimpleName() : message;
    }

    @Override public void onDestroy() {
        if (!stopping.get()) stopCapture("service_stopped", "Android stopped the capture service.");
        super.onDestroy();
    }

    @Override public IBinder onBind(Intent intent) { return null; }
}
