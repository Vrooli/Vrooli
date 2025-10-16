const DeviceVisionFilterDefs = () => (
  <svg className="device-emulation-vision-defs" aria-hidden focusable="false">
    <defs>
      <filter id="device-vision-protanopia">
        <feColorMatrix
          type="matrix"
          values="0.56667 0.43333 0 0 0 0.55833 0.44167 0 0 0 0 0.24167 0.75833 0 0 0 0 0 1 0"
        />
      </filter>
      <filter id="device-vision-deuteranopia">
        <feColorMatrix
          type="matrix"
          values="0.625 0.375 0 0 0 0.7 0.3 0 0 0 0 0.3 0.7 0 0 0 0 0 1 0"
        />
      </filter>
      <filter id="device-vision-tritanopia">
        <feColorMatrix
          type="matrix"
          values="0.95 0.05 0 0 0 0 0.43333 0.56667 0 0 0 0.475 0.525 0 0 0 0 0 1 0"
        />
      </filter>
    </defs>
  </svg>
);

export default DeviceVisionFilterDefs;
