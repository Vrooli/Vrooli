# PyBullet Resource Problems and Solutions

## Installation Issues

### Problem: python3-venv package not installed (FIXED)
**Symptoms**: Error "The virtual environment was not created successfully because ensurepip is not available"
**Solution**: Implemented fallback installation method using `pip install --target` that works without venv module
**Status**: ✅ Fixed in v2.0 - resource automatically detects and uses fallback method

### Problem: pip not found
**Symptoms**: Cannot install Python packages
**Solution**: Check for pip3 or python3 -m pip, provide clear error message if neither available
**Status**: ✅ Handled with clear error messages

### Problem: Permission denied on wrapper scripts
**Symptoms**: `/home/.../pybullet/.venv/bin/python: Permission denied`
**Solution**: Ensure chmod +x is applied to all wrapper scripts
**Status**: ✅ Fixed - scripts are now properly marked executable

## Known Limitations

1. **No GPU acceleration**: Current implementation uses CPU-only PyBullet
2. **Limited concurrent simulations**: Maximum 10 simulations to prevent resource exhaustion
3. **No VR support**: VR controller integration not implemented in initial version
4. **Basic API only**: Advanced features like soft body dynamics not exposed via API yet
5. **Physics constraint issues**: Point-to-point constraints may not behave as expected in some cases
6. **Timestep precision**: Real-time control needs further refinement for precise timing

## Common Errors

### Port Already in Use
**Error**: "Address already in use"
**Fix**: Check for existing processes on port 11457 with `lsof -i :11457`

### Import Error
**Error**: "No module named 'pybullet'"
**Fix**: Ensure virtual environment is activated and packages installed

### Health Check Timeout
**Error**: Health endpoint not responding
**Fix**: Check server logs with `vrooli resource pybullet logs`

## Performance Considerations

- Each simulation consumes ~50MB RAM
- Physics calculations are CPU-intensive
- Headless mode is 2-3x faster than GUI mode
- Network latency affects API response times

## Future Improvements

- Add GPU acceleration support
- Implement distributed simulation capabilities
- Add WebGL-based browser visualization
- Enhance soft body and cloth simulation
- Add ROS 2 integration