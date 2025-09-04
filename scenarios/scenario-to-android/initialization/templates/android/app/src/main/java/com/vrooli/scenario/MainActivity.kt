package com.vrooli.scenario.{{SCENARIO_NAME_PACKAGE}}

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.util.Log
import android.webkit.*
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import org.json.JSONObject
import java.io.File

class MainActivity : AppCompatActivity() {
    private lateinit var webView: WebView
    private val TAG = "VrooliApp"
    private val PERMISSION_REQUEST_CODE = 100
    
    private var filePathCallback: ValueCallback<Array<Uri>>? = null
    private val FILE_CHOOSER_RESULT_CODE = 200

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        // Create WebView programmatically for better control
        webView = WebView(this).apply {
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
            )
        }
        setContentView(webView)
        
        setupWebView()
        loadScenario()
    }
    
    private fun setupWebView() {
        webView.settings.apply {
            javaScriptEnabled = true
            domStorageEnabled = true
            allowFileAccess = true
            allowContentAccess = true
            mixedContentMode = WebSettings.MIXED_CONTENT_ALWAYS_ALLOW
            cacheMode = WebSettings.LOAD_DEFAULT
            
            // Enable database and local storage
            databaseEnabled = true
            
            // Enable app cache
            setAppCacheEnabled(true)
            setAppCachePath(cacheDir.absolutePath)
            
            // Enable zooming for better mobile experience
            setSupportZoom(true)
            builtInZoomControls = true
            displayZoomControls = false
            
            // Improve rendering performance
            setRenderPriority(WebSettings.RenderPriority.HIGH)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT) {
                setLayerType(View.LAYER_TYPE_HARDWARE, null)
            }
        }
        
        // Add JavaScript interface for native functionality
        webView.addJavascriptInterface(VrooliJSInterface(this), "VrooliNative")
        
        // Setup WebView client for handling navigation
        webView.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean {
                request?.url?.let { url ->
                    if (url.scheme in listOf("http", "https", "file")) {
                        return false // Let WebView handle it
                    }
                }
                return true
            }
            
            override fun onPageFinished(view: WebView?, url: String?) {
                super.onPageFinished(view, url)
                Log.d(TAG, "Page loaded: $url")
                
                // Inject service worker if needed
                injectServiceWorker()
            }
            
            override fun onReceivedError(view: WebView?, request: WebResourceRequest?, error: WebResourceError?) {
                super.onReceivedError(view, request, error)
                Log.e(TAG, "WebView error: ${error?.description}")
            }
        }
        
        // Setup Chrome client for advanced features
        webView.webChromeClient = object : WebChromeClient() {
            override fun onConsoleMessage(consoleMessage: ConsoleMessage): Boolean {
                Log.d(TAG, "${consoleMessage.message()} -- From line ${consoleMessage.lineNumber()}")
                return true
            }
            
            override fun onPermissionRequest(request: PermissionRequest?) {
                request?.grant(request.resources)
            }
            
            override fun onShowFileChooser(
                webView: WebView?,
                filePathCallback: ValueCallback<Array<Uri>>?,
                fileChooserParams: FileChooserParams?
            ): Boolean {
                this@MainActivity.filePathCallback = filePathCallback
                
                val intent = Intent(Intent.ACTION_GET_CONTENT)
                intent.type = "*/*"
                intent.addCategory(Intent.CATEGORY_OPENABLE)
                
                startActivityForResult(
                    Intent.createChooser(intent, "Choose File"),
                    FILE_CHOOSER_RESULT_CODE
                )
                
                return true
            }
        }
        
        // Enable debugging in debug builds
        if (BuildConfig.DEBUG) {
            WebView.setWebContentsDebuggingEnabled(true)
        }
    }
    
    private fun loadScenario() {
        // Check if we have offline assets
        val indexFile = File(filesDir, "index.html")
        
        if (indexFile.exists()) {
            // Load from local storage (offline mode)
            webView.loadUrl("file://${indexFile.absolutePath}")
        } else {
            // Load from assets (first run)
            webView.loadUrl("file:///android_asset/index.html")
        }
    }
    
    private fun injectServiceWorker() {
        val js = """
            if ('serviceWorker' in navigator) {
                navigator.serviceWorker.register('/sw.js')
                    .then(reg => console.log('Service Worker registered'))
                    .catch(err => console.error('Service Worker registration failed:', err));
            }
        """.trimIndent()
        
        webView.evaluateJavascript(js, null)
    }
    
    override fun onBackPressed() {
        if (webView.canGoBack()) {
            webView.goBack()
        } else {
            super.onBackPressed()
        }
    }
    
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        
        if (requestCode == FILE_CHOOSER_RESULT_CODE) {
            if (resultCode == RESULT_OK && data != null) {
                val result = data.data?.let { arrayOf(it) }
                filePathCallback?.onReceiveValue(result)
            } else {
                filePathCallback?.onReceiveValue(null)
            }
            filePathCallback = null
        }
    }
    
    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        
        if (requestCode == PERMISSION_REQUEST_CODE) {
            permissions.forEachIndexed { index, permission ->
                val granted = grantResults[index] == PackageManager.PERMISSION_GRANTED
                Log.d(TAG, "$permission: ${if (granted) "GRANTED" else "DENIED"}")
                
                // Notify JavaScript about permission result
                val js = "window.onPermissionResult && window.onPermissionResult('$permission', $granted);"
                webView.evaluateJavascript(js, null)
            }
        }
    }
    
    override fun onDestroy() {
        webView.loadUrl("about:blank")
        webView.destroy()
        super.onDestroy()
    }
}

/**
 * JavaScript interface providing native Android functionality to the WebView
 */
class VrooliJSInterface(private val context: Context) {
    
    @JavascriptInterface
    fun getDeviceInfo(): String {
        return JSONObject().apply {
            put("platform", "android")
            put("version", Build.VERSION.SDK_INT)
            put("versionName", Build.VERSION.RELEASE)
            put("model", Build.MODEL)
            put("manufacturer", Build.MANUFACTURER)
            put("brand", Build.BRAND)
        }.toString()
    }
    
    @JavascriptInterface
    fun showToast(message: String) {
        (context as? MainActivity)?.runOnUiThread {
            Toast.makeText(context, message, Toast.LENGTH_SHORT).show()
        }
    }
    
    @JavascriptInterface
    fun requestPermission(permission: String) {
        val activity = context as? MainActivity ?: return
        
        val androidPermission = when(permission) {
            "camera" -> Manifest.permission.CAMERA
            "location" -> Manifest.permission.ACCESS_FINE_LOCATION
            "storage" -> if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                Manifest.permission.READ_MEDIA_IMAGES
            } else {
                Manifest.permission.READ_EXTERNAL_STORAGE
            }
            else -> return
        }
        
        if (ContextCompat.checkSelfPermission(activity, androidPermission) 
            != PackageManager.PERMISSION_GRANTED) {
            ActivityCompat.requestPermissions(
                activity, 
                arrayOf(androidPermission), 
                activity.PERMISSION_REQUEST_CODE
            )
        }
    }
    
    @JavascriptInterface
    fun hasPermission(permission: String): Boolean {
        val androidPermission = when(permission) {
            "camera" -> Manifest.permission.CAMERA
            "location" -> Manifest.permission.ACCESS_FINE_LOCATION
            "storage" -> if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                Manifest.permission.READ_MEDIA_IMAGES
            } else {
                Manifest.permission.READ_EXTERNAL_STORAGE
            }
            else -> return false
        }
        
        return ContextCompat.checkSelfPermission(context, androidPermission) == 
               PackageManager.PERMISSION_GRANTED
    }
    
    @JavascriptInterface
    fun saveData(key: String, value: String) {
        val sharedPref = context.getSharedPreferences("VrooliData", Context.MODE_PRIVATE)
        with(sharedPref.edit()) {
            putString(key, value)
            apply()
        }
    }
    
    @JavascriptInterface
    fun getData(key: String): String? {
        val sharedPref = context.getSharedPreferences("VrooliData", Context.MODE_PRIVATE)
        return sharedPref.getString(key, null)
    }
    
    @JavascriptInterface
    fun vibrate(duration: Long) {
        val vibrator = context.getSystemService(Context.VIBRATOR_SERVICE) as? android.os.Vibrator
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            vibrator?.vibrate(
                android.os.VibrationEffect.createOneShot(
                    duration, 
                    android.os.VibrationEffect.DEFAULT_AMPLITUDE
                )
            )
        } else {
            @Suppress("DEPRECATION")
            vibrator?.vibrate(duration)
        }
    }
}